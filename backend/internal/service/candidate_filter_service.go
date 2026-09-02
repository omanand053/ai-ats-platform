package service

import (
	"context"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/utils"

	"github.com/google/uuid"
)

type candidateFilterCandidateRepo interface {
	ListByJobID(ctx context.Context, companyID, jobID uuid.UUID) ([]domain.Candidate, error)
	ListForSemanticFiltering(ctx context.Context, companyID uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error)
}

type candidateFilterJobRepo interface {
	GetByID(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error)
}

// CandidateFilterWeights controls how strongly each pre-filter dimension contributes.
type CandidateFilterWeights struct {
	Role       float64
	Skills     float64
	Experience float64
}

func DefaultCandidateFilterWeights() CandidateFilterWeights {
	return CandidateFilterWeights{Role: 0.3, Skills: 0.5, Experience: 0.2}
}

func normalizeCandidateFilterWeights(weights CandidateFilterWeights) CandidateFilterWeights {
	sum := weights.Role + weights.Skills + weights.Experience
	if sum <= 0 {
		return DefaultCandidateFilterWeights()
	}
	return CandidateFilterWeights{
		Role:       weights.Role / sum,
		Skills:     weights.Skills / sum,
		Experience: weights.Experience / sum,
	}
}

// CandidateFilterService scores candidates before semantic search using role, skills,
// and experience heuristics. It returns a shortlist of candidate IDs to run semantic
// search against.
type CandidateFilterService struct {
	candidateRepo candidateFilterCandidateRepo
	jobRepo       candidateFilterJobRepo
	weights       CandidateFilterWeights
	threshold     float64
	roleSynonyms  map[string][]string
}

func NewCandidateFilterService(
	candidateRepo *repository.CandidateRepository,
	jobRepo *repository.JobRepository,
	weights CandidateFilterWeights,
	threshold float64,
) *CandidateFilterService {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	return &CandidateFilterService{
		candidateRepo: candidateRepo,
		jobRepo:       jobRepo,
		weights:       normalizeCandidateFilterWeights(weights),
		threshold:     threshold,
		roleSynonyms:  defaultRoleSynonyms(),
	}
}

func (s *CandidateFilterService) SoftThreshold() float64 {
	if s == nil {
		return 40
	}
	return s.threshold
}

// ApplicantIDsForJob returns every applicant assigned to the job (candidates.job_id).
// It never searches the company-wide pool and never applies an eligibility hard-gate.
func (s *CandidateFilterService) ApplicantIDsForJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
) ([]uuid.UUID, error) {
	if s == nil || s.candidateRepo == nil {
		return nil, nil
	}
	applicants, err := s.candidateRepo.ListByJobID(ctx, companyID, jobID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(applicants))
	for i := range applicants {
		ids = append(ids, applicants[i].ID)
	}
	return ids, nil
}

func (s *CandidateFilterService) EligibleCandidateIDsForJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	candidateIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	eligible, _, err := s.EvaluateApplicantsForJob(ctx, companyID, jobID, candidateIDs)
	return eligible, err
}

// EvaluateApplicantsForJob soft-scores applicants and reports who fall below the
// eligibility threshold. Semantic ranking should use ApplicantIDsForJob instead —
// eligibility must not hard-block ranking.
func (s *CandidateFilterService) EvaluateApplicantsForJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	candidateIDs []uuid.UUID,
) ([]uuid.UUID, []domain.NotShortlistedCandidate, error) {
	if s == nil || s.candidateRepo == nil || s.jobRepo == nil {
		return nil, nil, nil
	}

	job, err := s.jobRepo.GetByID(ctx, companyID, jobID)
	if err != nil {
		return nil, nil, err
	}

	// Semantic matching is always scoped to this job's applicants (candidates.job_id).
	filterIDs := candidateIDs
	if len(filterIDs) == 0 {
		applicants, err := s.candidateRepo.ListByJobID(ctx, companyID, jobID)
		if err != nil {
			return nil, nil, err
		}
		if len(applicants) == 0 {
			return []uuid.UUID{}, nil, nil
		}
		filterIDs = make([]uuid.UUID, 0, len(applicants))
		for i := range applicants {
			filterIDs = append(filterIDs, applicants[i].ID)
		}
	}

	candidates, err := s.candidateRepo.ListForSemanticFiltering(ctx, companyID, filterIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(candidates) == 0 {
		return []uuid.UUID{}, nil, nil
	}

	eligible := make([]uuid.UUID, 0, len(candidates))
	rejected := make([]domain.NotShortlistedCandidate, 0)
	type scored struct {
		id    uuid.UUID
		score float64
	}
	ranked := make([]scored, 0, len(candidates))

	for i := range candidates {
		c := &candidates[i]
		score := s.ScoreCandidate(c, job)
		if score >= s.threshold {
			ranked = append(ranked, scored{id: c.ID, score: score})
			continue
		}
		reason := eligibilityRejectReason(c, job, score, s.threshold, s)
		scoreCopy := score
		rejected = append(rejected, domain.NotShortlistedCandidate{
			CandidateID:       c.ID,
			CandidateName:     c.Name,
			Reason:            "failed_eligibility",
			EligibilityScore:  &scoreCopy,
			WhyNotShortlisted: reason,
		})
	}

	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	for _, item := range ranked {
		eligible = append(eligible, item.id)
	}
	return eligible, rejected, nil
}

func eligibilityRejectReason(
	candidate *domain.Candidate,
	job *domain.Job,
	score, threshold float64,
	s *CandidateFilterService,
) string {
	if candidate == nil || job == nil {
		return "Candidate could not be evaluated for eligibility."
	}
	if s != nil && !s.roleMatchesCandidate(candidate, job.Title) {
		designation := "unknown role"
		if candidate.CurrentDesignation != nil && strings.TrimSpace(*candidate.CurrentDesignation) != "" {
			designation = strings.TrimSpace(*candidate.CurrentDesignation)
		}
		return fmt.Sprintf(
			"Not shortlisted: role mismatch between candidate (%s) and job (%s). Eligibility score %.0f%% is below the %.0f%% gate.",
			designation, job.Title, score, threshold,
		)
	}
	return fmt.Sprintf(
		"Not shortlisted: eligibility score %.0f%% is below the %.0f%% pre-filter threshold (skills/experience).",
		score, threshold,
	)
}

func (s *CandidateFilterService) ScoreCandidate(candidate *domain.Candidate, job *domain.Job) float64 {
	if s == nil {
		return 0
	}
	if candidate == nil || job == nil {
		return 0
	}
	if !s.roleMatchesCandidate(candidate, job.Title) {
		return 0
	}
	weights := normalizeCandidateFilterWeights(s.weights)
	skillsScore := scoreSkills(candidate.Skills, job.RequiredSkills)
	experienceScore := scoreExperience(candidate.ExperienceYears, job.ExperienceRequired)

	return skillsScore*weights.Skills + experienceScore*weights.Experience
}

func (s *CandidateFilterService) roleMatches(candidateDesignation *string, jobTitle string) bool {
	if s == nil {
		return false
	}
	if candidateDesignation == nil || strings.TrimSpace(*candidateDesignation) == "" {
		return false
	}
	jobFamily := roleFamilyKey(jobTitle)
	candidateFamily := roleFamilyKey(*candidateDesignation)
	if jobFamily == "" || candidateFamily == "" {
		return false
	}
	if jobFamily == candidateFamily {
		return true
	}
	if s.matchesSynonym(candidateFamily, jobFamily) {
		return true
	}
	return false
}

func (s *CandidateFilterService) roleMatchesCandidate(candidate *domain.Candidate, jobTitle string) bool {
	if s == nil || candidate == nil {
		return false
	}
	if s.roleMatches(candidate.CurrentDesignation, jobTitle) {
		return true
	}

	for _, hint := range []string{
		candidateResumeHint(candidate.ResumeSummary),
		candidateResumeHint(candidate.ResumeText),
	} {
		if strings.TrimSpace(hint) == "" {
			continue
		}
		if s.roleMatches(&hint, jobTitle) {
			return true
		}
	}
	return false
}

func candidateResumeHint(value *string) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "@") || strings.Contains(lower, "http") {
			continue
		}
		if strings.Contains(lower, "summary") || strings.Contains(lower, "skills") || strings.Contains(lower, "experience") || strings.Contains(lower, "education") || strings.Contains(lower, "project") {
			continue
		}
		if strings.Contains(lower, "developer") || strings.Contains(lower, "engineer") || strings.Contains(lower, "manager") || strings.Contains(lower, "analyst") || strings.Contains(lower, "designer") || strings.Contains(lower, "architect") || strings.Contains(lower, "scientist") || strings.Contains(lower, "consultant") || strings.Contains(lower, "specialist") || strings.Contains(lower, "administrator") || strings.Contains(lower, "lead") || strings.Contains(lower, "intern") || strings.Contains(lower, "programmer") || strings.Contains(lower, "officer") || strings.Contains(lower, "recruiter") {
			return line
		}
	}
	return ""
}

func (s *CandidateFilterService) matchesSynonym(candidateFamily, jobFamily string) bool {
	if s == nil {
		return false
	}
	if candidateFamily == "" || jobFamily == "" {
		return false
	}
	for _, synonymGroup := range s.roleSynonyms {
		groupSet := make(map[string]struct{}, len(synonymGroup))
		for _, synonym := range synonymGroup {
			groupSet[roleFamilyKey(synonym)] = struct{}{}
		}
		if _, ok := groupSet[jobFamily]; !ok {
			continue
		}
		if _, ok := groupSet[candidateFamily]; ok {
			return true
		}
	}
	return false
}

func defaultRoleSynonyms() map[string][]string {
	return map[string][]string{
		"backend": {"backend engineer", "software engineer", "go developer", "sde backend"},
		"hr":      {"hr executive", "recruiter", "talent acquisition", "hr generalist"},
		"ai":      {"ai engineer", "ml engineer", "data scientist"},
	}
}

func scoreRole(candidateDesignation *string, jobTitle string) float64 {
	if candidateDesignation == nil || strings.TrimSpace(*candidateDesignation) == "" {
		return 0
	}
	jobFamily := roleFamilyKey(jobTitle)
	candidateFamily := roleFamilyKey(*candidateDesignation)
	if jobFamily == "" || candidateFamily == "" {
		return 0
	}
	if jobFamily == candidateFamily {
		return 100
	}
	return 0
}

func scoreSkills(candidateSkills, requiredSkills []string) float64 {
	if len(requiredSkills) == 0 {
		return 100
	}
	candKeys := make(map[string]struct{}, len(candidateSkills))
	for _, skill := range candidateSkills {
		key := utils.NormalizeSkillKey(skill)
		if key == "" {
			continue
		}
		candKeys[key] = struct{}{}
	}
	matched := 0
	for _, skill := range requiredSkills {
		key := utils.NormalizeSkillKey(skill)
		if _, ok := candKeys[key]; ok {
			matched++
		}
	}
	return clamp(float64(matched)/float64(len(requiredSkills))*100, 0, 100)
}

func scoreExperience(candidateYears *int, experienceRequired *string) float64 {
	required := utils.ParseRequiredYears(experienceRequired)
	if required == nil || *required <= 0 {
		return 100
	}
	if candidateYears == nil {
		return 0
	}
	if float64(*candidateYears) >= *required {
		return 100
	}
	ratio := float64(*candidateYears) / *required
	return clamp(ratio*100, 0, 100)
}

func normalizeRoleText(text string) string {
	return roleFamilyKey(text)
}

func roleTokens(text string) []string {
	clean := strings.ToLower(strings.TrimSpace(text))
	clean = strings.ReplaceAll(clean, ",", " ")
	clean = strings.ReplaceAll(clean, "/", " ")
	clean = strings.ReplaceAll(clean, "-", " ")
	parts := strings.Fields(clean)
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if isGenericRoleToken(part) {
			continue
		}
		filtered = append(filtered, part)
	}
	return filtered
}

func isGenericRoleToken(token string) bool {
	genericTokens := map[string]struct{}{
		"senior":     {},
		"junior":     {},
		"executive":  {},
		"engineer":   {},
		"developer":  {},
		"associate":  {},
		"assistant":  {},
		"lead":       {},
		"manager":    {},
		"specialist": {},
		"principal":  {},
		"staff":      {},
		"analyst":    {},
		"consultant": {},
	}
	_, ok := genericTokens[token]
	return ok
}

func roleFamilyKey(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.NewReplacer(",", " ", "/", " ", "-", " ", "&", " ").Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	for _, candidate := range []struct {
		family string
		terms  []string
	}{
		{family: "sales", terms: []string{"sales", "business development", "businessdevelopment", "bd", "account"}},
		{family: "marketing", terms: []string{"marketing", "digital marketing", "seo", "performance marketing", "growth", "content", "social"}},
		{family: "hr", terms: []string{"hr", "human resource", "human resources", "talent", "recruit", "recruiter", "hiring", "people"}},
		{family: "engineering", terms: []string{"backend", "software", "service", "api", "platform", "systems", "system", "sde", "go", "java", "node", "python", "javascript", "typescript", "c++", "php", "ruby", "kotlin", "scala", "dotnet"}},
		{family: "frontend", terms: []string{"frontend", "react", "ui", "ux", "web", "next", "vue", "angular", "css", "html"}},
		{family: "qa", terms: []string{"qa", "quality", "quality assurance", "test", "testing", "automation", "sdet", "playwright", "manual"}},
		{family: "devops", terms: []string{"devops", "sre", "site reliability", "cloud", "infra", "infrastructure", "kubernetes", "terraform", "docker", "linux", "ops"}},
		{family: "ai", terms: []string{"ai", "ml", "machine learning", "deep learning", "data science", "datascience", "nlp", "vision", "gen ai", "genai"}},
		{family: "data", terms: []string{"data analyst", "data engineer", "analytics", "bi", "reporting", "warehouse", "etl", "sql", "database"}},
		{family: "product", terms: []string{"product"}},
	} {
		for _, term := range candidate.terms {
			if strings.Contains(normalized, term) {
				return candidate.family
			}
		}
	}
	return strings.Join(roleTokens(normalized), "_")
}

func (s *CandidateFilterService) SetRoleSynonyms(roleSynonyms map[string][]string) error {
	if s == nil {
		return fmt.Errorf("candidate filter service is nil")
	}
	if roleSynonyms == nil {
		s.roleSynonyms = defaultRoleSynonyms()
		return nil
	}
	normalized := make(map[string][]string, len(roleSynonyms))
	for key, values := range roleSynonyms {
		if strings.TrimSpace(key) == "" {
			continue
		}
		seen := make(map[string]struct{}, len(values))
		group := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			group = append(group, trimmed)
		}
		if len(group) > 0 {
			normalized[strings.ToLower(strings.TrimSpace(key))] = group
		}
	}
	s.roleSynonyms = normalized
	return nil
}

func tokenSet(text string) map[string]struct{} {
	parts := strings.Fields(strings.TrimSpace(text))
	set := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		set[part] = struct{}{}
	}
	return set
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
