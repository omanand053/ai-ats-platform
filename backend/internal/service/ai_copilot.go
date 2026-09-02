package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/llm"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

type copilotCacheEntry struct {
	resp      *domain.AIAssistantResponse
	expiresAt time.Time
}

// CopilotRequest drives Phase 4 AI Recruiter Copilot panels.
type CopilotRequest struct {
	Type         string
	Question     string
	TopK         int
	CandidateID  *uuid.UUID
	CandidateIDs []uuid.UUID
	EmailKind    string
	Difficulty   string
}

func (s *AIAssistantService) SetCandidateRepo(repo *repository.CandidateRepository) {
	if s != nil {
		s.candidates = repo
	}
}

// Copilot runs a typed recruiter-copilot action. Backward-compatible Ask uses type=qa.
func (s *AIAssistantService) Copilot(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if typ == "" || typ == "qa" {
		return s.Ask(ctx, companyID, jobID, req.Question, req.TopK)
	}

	cacheKey := s.copilotCacheKey(companyID, jobID, req)
	if cached := s.getCopilotCache(cacheKey); cached != nil {
		out := *cached
		out.Cached = true
		return &out, nil
	}

	var (
		resp *domain.AIAssistantResponse
		err  error
	)
	switch typ {
	case "summary":
		resp, err = s.copilotSummary(ctx, companyID, jobID, req)
	case "interview":
		resp, err = s.copilotInterview(ctx, companyID, jobID, req)
	case "recommendation":
		resp, err = s.copilotRecommendation(ctx, companyID, jobID, req)
	case "insights":
		resp, err = s.copilotInsights(ctx, companyID, jobID, req)
	case "jd_optimizer":
		resp, err = s.copilotJDOptimizer(ctx, companyID, jobID)
	case "email":
		resp, err = s.copilotEmail(ctx, companyID, jobID, req)
	case "compare":
		resp, err = s.copilotCompare(ctx, companyID, jobID, req)
	default:
		return nil, fmt.Errorf("unsupported copilot type: %s", typ)
	}
	if err != nil {
		return nil, err
	}
	if resp != nil {
		resp.Type = typ
		if resp.AIAvailable {
			s.putCopilotCache(cacheKey, resp)
		}
	}
	return resp, nil
}

func (s *AIAssistantService) copilotSummary(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	job, candidate, match, err := s.loadCandidateContext(ctx, companyID, jobID, req.CandidateID)
	if err != nil {
		return nil, err
	}
	structured := deterministicSummary(candidate, match, job)
	explain := explainFromMatch(match, candidate)
	resp := baseCopilotResp(match, explain)
	resp.Structured = structured
	resp.Answer = formatSummaryAnswer(structured)

	prompt := fmt.Sprintf(
		`Write a concise recruiter brief for this candidate vs the job.
Include: Professional Summary, Strengths, Weaknesses, Career Growth, Risk Factors, Recommended Role, Seniority, Notice Period (say unknown if not present).
Use ONLY the context. Be factual.

JOB:
Title: %s
Skills: %s
Experience: %s

CANDIDATE:
Name: %s
Designation: %s
Experience years: %v
Skills: %s
Summary: %s
AI Match: %.0f%% Confidence: %s
Strengths: %s
Missing: %s
Resume excerpt:
%s`,
		job.Title,
		strings.Join(job.RequiredSkills, ", "),
		strPtr(job.ExperienceRequired),
		candidate.Name,
		strPtr(candidate.CurrentDesignation),
		intPtr(candidate.ExperienceYears),
		strings.Join(candidate.Skills, ", "),
		strPtr(candidate.ResumeSummary),
		matchScore(match),
		matchConfidence(match),
		strings.Join(matchStrengths(match), "; "),
		strings.Join(matchMissing(match), ", "),
		truncate(strPtr(candidate.ResumeText), 2200),
	)
	return s.enrichWithLLM(ctx, resp, copilotSystem(), prompt, "Produce the recruiter brief.")
}

func (s *AIAssistantService) copilotInterview(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	job, candidate, match, err := s.loadCandidateContext(ctx, companyID, jobID, req.CandidateID)
	if err != nil {
		return nil, err
	}
	diff := strings.ToLower(strings.TrimSpace(req.Difficulty))
	if diff == "" {
		diff = "medium"
	}
	explain := explainFromMatch(match, candidate)
	resp := baseCopilotResp(match, explain)
	resp.Structured = map[string]any{"difficulty": diff}

	prompt := fmt.Sprintf(
		`Generate an interview kit for difficulty=%s.
Sections:
1) Technical Questions (5)
2) Behavioral Questions (4)
3) Project Questions (3)
4) Scenario Questions (3)
5) Follow-up Questions (3)
For each question include a one-line what-good-looks-like note.
Tailor to the job and candidate gaps.

JOB: %s | skills=%s
CANDIDATE: %s | skills=%s | missing=%s | projects/resume:
%s`,
		diff,
		job.Title,
		strings.Join(job.RequiredSkills, ", "),
		candidate.Name,
		strings.Join(candidate.Skills, ", "),
		strings.Join(matchMissing(match), ", "),
		truncate(strPtr(candidate.ResumeText), 1800),
	)
	return s.enrichWithLLM(ctx, resp, copilotSystem(), prompt, "Generate the interview kit.")
}

func (s *AIAssistantService) copilotRecommendation(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	_, candidate, match, err := s.loadCandidateContext(ctx, companyID, jobID, req.CandidateID)
	if err != nil {
		return nil, err
	}
	rec, reason := hiringBand(match)
	explain := explainFromMatch(match, candidate)
	explain.Reason = reason
	resp := baseCopilotResp(match, explain)
	resp.Structured = map[string]any{
		"recommendation": rec,
		"explanation":    reason,
		"ai_match":       matchScore(match),
		"confidence":     matchConfidence(match),
	}
	resp.Answer = fmt.Sprintf("Recommendation: %s\n\n%s", rec, reason)

	prompt := fmt.Sprintf(
		`Given recommendation=%s and evidence, write a short hiring memo (4-6 sentences) with clear next step.
Candidate=%s AI Match=%.0f Confidence=%s Strengths=%s Missing=%s`,
		rec,
		candidate.Name,
		matchScore(match),
		matchConfidence(match),
		strings.Join(matchStrengths(match), "; "),
		strings.Join(matchMissing(match), ", "),
	)
	return s.enrichWithLLM(ctx, resp, copilotSystem(), prompt, "Write the hiring memo.")
}

func (s *AIAssistantService) copilotInsights(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	_, candidate, match, err := s.loadCandidateContext(ctx, companyID, jobID, req.CandidateID)
	if err != nil {
		return nil, err
	}
	insights := resumeInsights(candidate, match)
	explain := explainFromMatch(match, candidate)
	resp := baseCopilotResp(match, explain)
	resp.Structured = insights
	resp.Answer = formatInsightsAnswer(insights)
	resp.AIAvailable = true
	resp.ProviderUsed = "deterministic"
	resp.Provider = "deterministic"
	return resp, nil
}

func (s *AIAssistantService) copilotJDOptimizer(
	ctx context.Context,
	companyID, jobID uuid.UUID,
) (*domain.AIAssistantResponse, error) {
	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	resp := &domain.AIAssistantResponse{
		ReferencedCandidates: []domain.AIAssistantReferencedCandidate{},
		SemanticMatchesUsed:  []domain.SemanticMatch{},
		Explainability: &domain.AIExplainability{
			Reason:                 "JD analysis based on the current job requisition text and required skills.",
			Evidence:               []string{job.Title, strings.Join(job.RequiredSkills, ", ")},
			Confidence:             "medium",
			RelevantResumeSections: []string{},
		},
	}
	prompt := fmt.Sprintf(
		`Optimize this job description for ATS and recruiter clarity.
Suggest:
- Missing skills
- Better wording
- Industry keywords
- Skill redundancy to remove
- Improved short JD rewrite

Title: %s
Location: %s
Experience: %s
Required skills: %s
Description:
%s`,
		job.Title, strPtr(job.Location), strPtr(job.ExperienceRequired),
		strings.Join(job.RequiredSkills, ", "),
		truncate(strPtr(job.Description), 2500),
	)
	return s.enrichWithLLM(ctx, resp, copilotSystem(), prompt, "Optimize the JD.")
}

func (s *AIAssistantService) copilotEmail(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	job, candidate, match, err := s.loadCandidateContext(ctx, companyID, jobID, req.CandidateID)
	if err != nil {
		return nil, err
	}
	kind := strings.ToLower(strings.TrimSpace(req.EmailKind))
	if kind == "" {
		kind = "interview_invite"
	}
	explain := explainFromMatch(match, candidate)
	resp := baseCopilotResp(match, explain)
	resp.Structured = map[string]any{"email_kind": kind}

	prompt := fmt.Sprintf(
		`Write a professional recruiter email of type=%s for this job/candidate.
Include Subject and Body. Keep it concise and respectful.

Job: %s
Candidate: %s (%s)
Company role context skills: %s`,
		kind,
		job.Title,
		candidate.Name,
		candidate.Email,
		strings.Join(job.RequiredSkills, ", "),
	)
	return s.enrichWithLLM(ctx, resp, copilotSystem(), prompt, "Write the email.")
}

func (s *AIAssistantService) copilotCompare(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	req CopilotRequest,
) (*domain.AIAssistantResponse, error) {
	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	ids := req.CandidateIDs
	if len(ids) == 0 && req.CandidateID != nil {
		ids = []uuid.UUID{*req.CandidateID}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("candidate_ids required for compare")
	}
	if len(ids) > 5 {
		ids = ids[:5]
	}

	matchResult, _, err := s.retriever.Retrieve(ctx, companyID, jobID, 50)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]domain.SemanticMatch{}
	if matchResult != nil {
		for _, m := range matchResult.Matches {
			byID[m.CandidateID] = m
		}
	}

	rows := make([]map[string]any, 0, len(ids))
	var winnerID uuid.UUID
	winnerScore := -1.0
	for _, id := range ids {
		c, cerr := s.candidates.GetByID(ctx, companyID, id)
		if cerr != nil {
			continue
		}
		m := byID[id]
		score := matchScore(&m)
		if score > winnerScore {
			winnerScore = score
			winnerID = id
		}
		rows = append(rows, map[string]any{
			"candidate_id":   id.String(),
			"candidate_name": c.Name,
			"ai_match":       score,
			"semantic":       m.SimilarityScore,
			"skills":         breakdownVal(m, "skills"),
			"experience":     breakdownVal(m, "experience"),
			"projects":       breakdownVal(m, "projects"),
			"education":      breakdownVal(m, "education"),
			"strengths":      matchStrengths(&m),
			"weaknesses":     matchMissing(&m),
			"confidence":     matchConfidence(&m),
			"recommendation": func() string { r, _ := hiringBand(&m); return r }(),
		})
	}

	structured := map[string]any{
		"candidates": rows,
		"winner_id":  winnerID.String(),
		"job_title":  job.Title,
	}
	resp := &domain.AIAssistantResponse{
		AIAvailable:          true,
		ProviderUsed:         "deterministic",
		Provider:             "deterministic",
		ReferencedCandidates: []domain.AIAssistantReferencedCandidate{},
		SemanticMatchesUsed:  []domain.SemanticMatch{},
		Structured:           structured,
		Answer:               formatCompareAnswer(rows, winnerID),
		Explainability: &domain.AIExplainability{
			Reason:                 "Winner selected by highest Overall AI Match among compared applicants.",
			Evidence:               []string{fmt.Sprintf("winner_score=%.0f", winnerScore)},
			Confidence:             "high",
			RelevantResumeSections: []string{"skills", "experience", "projects"},
		},
	}
	_ = ctx
	return resp, nil
}

func (s *AIAssistantService) loadCandidateContext(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	candidateID *uuid.UUID,
) (*domain.Job, *domain.Candidate, *domain.SemanticMatch, error) {
	if candidateID == nil {
		return nil, nil, nil, fmt.Errorf("candidate_id is required")
	}
	if s.candidates == nil {
		return nil, nil, nil, fmt.Errorf("candidate repository not configured")
	}
	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, nil, nil, ErrJobNotFound
		}
		return nil, nil, nil, err
	}
	candidate, err := s.candidates.GetByID(ctx, companyID, *candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, nil, nil, ErrCandidateNotFound
		}
		return nil, nil, nil, err
	}
	match := &domain.SemanticMatch{
		CandidateID:   candidate.ID,
		CandidateName: candidate.Name,
		Strengths:     candidate.MatchedSkills,
		MissingSkills: candidate.MissingSkills,
	}
	if matchResult, _, rerr := s.retriever.Retrieve(ctx, companyID, jobID, 50); rerr == nil && matchResult != nil {
		for i := range matchResult.Matches {
			if matchResult.Matches[i].CandidateID == candidate.ID {
				m := matchResult.Matches[i]
				match = &m
				break
			}
		}
	}
	return job, candidate, match, nil
}

func (s *AIAssistantService) enrichWithLLM(
	ctx context.Context,
	resp *domain.AIAssistantResponse,
	systemPrompt, userPrompt, question string,
) (*domain.AIAssistantResponse, error) {
	if reason := s.geminiUnavailableReason(); reason != "" {
		return s.unavailable(resp, reason), nil
	}
	gemini := llm.NewGeminiProvider(s.geminiKey, s.geminiModel)
	if err := gemini.VerifyModel(ctx); err != nil {
		return s.unavailable(resp, classifyGeminiFallbackReason(err)), nil
	}
	gen, err := gemini.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Question:     question,
		Context:      userPrompt,
	})
	if err != nil {
		return s.unavailable(resp, classifyGeminiFallbackReason(err)), nil
	}
	resp.AIAvailable = true
	resp.ProviderUsed = "gemini"
	resp.Provider = gen.Provider
	resp.Model = gen.Model
	resp.Answer = gen.Answer
	resp.Message = ""
	resp.FallbackReason = ""
	return resp, nil
}

func (s *AIAssistantService) copilotCacheKey(companyID, jobID uuid.UUID, req CopilotRequest) string {
	parts := []string{
		companyID.String(),
		jobID.String(),
		strings.ToLower(req.Type),
		strings.ToLower(req.EmailKind),
		strings.ToLower(req.Difficulty),
		req.Question,
	}
	if req.CandidateID != nil {
		parts = append(parts, req.CandidateID.String())
	}
	for _, id := range req.CandidateIDs {
		parts = append(parts, id.String())
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func (s *AIAssistantService) getCopilotCache(key string) *domain.AIAssistantResponse {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if s.cache == nil {
		return nil
	}
	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	cp := *entry.resp
	return &cp
}

func (s *AIAssistantService) putCopilotCache(key string, resp *domain.AIAssistantResponse) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache == nil {
		s.cache = map[string]copilotCacheEntry{}
	}
	cp := *resp
	s.cache[key] = copilotCacheEntry{resp: &cp, expiresAt: time.Now().Add(10 * time.Minute)}
}

// --- helpers ---

func copilotSystem() string {
	return `You are an AI Recruiter Copilot inside an ATS.
Be concise, professional, and evidence-based.
Never invent employers, degrees, or skills not present in the context.
When uncertain, say so.`
}

func baseCopilotResp(match *domain.SemanticMatch, explain *domain.AIExplainability) *domain.AIAssistantResponse {
	resp := &domain.AIAssistantResponse{
		ReferencedCandidates: []domain.AIAssistantReferencedCandidate{},
		SemanticMatchesUsed:  []domain.SemanticMatch{},
		Explainability:       explain,
	}
	if match != nil {
		resp.SemanticMatchesUsed = []domain.SemanticMatch{*match}
		conf := match.SimilarityScore
		resp.ConfidenceScore = &conf
	}
	return resp
}

func explainFromMatch(match *domain.SemanticMatch, candidate *domain.Candidate) *domain.AIExplainability {
	evidence := []string{}
	sections := []string{}
	if match != nil {
		evidence = append(evidence, fmt.Sprintf("Overall AI Match %.0f%%", match.AIMatchScore))
		evidence = append(evidence, fmt.Sprintf("Semantic similarity %.0f%%", match.SimilarityScore))
		if len(match.MatchedSkills) > 0 {
			evidence = append(evidence, "Matched skills: "+strings.Join(match.MatchedSkills, ", "))
			sections = append(sections, "skills")
		}
		if len(match.MissingSkills) > 0 {
			evidence = append(evidence, "Missing skills: "+strings.Join(match.MissingSkills, ", "))
		}
		if match.WhyShortlisted != "" {
			evidence = append(evidence, match.WhyShortlisted)
		}
	}
	if candidate != nil && candidate.ResumeSummary != nil && strings.TrimSpace(*candidate.ResumeSummary) != "" {
		sections = append(sections, "summary")
	}
	conf := matchConfidence(match)
	return &domain.AIExplainability{
		Reason:                 "Derived from Overall AI Match dimensions and resume evidence for this job.",
		Evidence:               evidence,
		Confidence:             conf,
		RelevantResumeSections: sections,
	}
}

func hiringBand(match *domain.SemanticMatch) (string, string) {
	score := matchScore(match)
	missing := len(matchMissing(match))
	switch {
	case score >= 85 && missing <= 1:
		return "Strong Hire", fmt.Sprintf("AI Match %.0f%% with minimal skill gaps — strong fit for this requisition.", score)
	case score >= 70:
		return "Hire", fmt.Sprintf("AI Match %.0f%% indicates solid alignment; proceed with structured interview.", score)
	case score >= 55:
		return "Borderline", fmt.Sprintf("AI Match %.0f%% with notable gaps; interview only if pipeline is thin.", score)
	default:
		return "Reject", fmt.Sprintf("AI Match %.0f%% is below the hiring bar for this role.", score)
	}
}

func deterministicSummary(candidate *domain.Candidate, match *domain.SemanticMatch, job *domain.Job) map[string]any {
	seniority := "Mid"
	if candidate.ExperienceYears != nil {
		if *candidate.ExperienceYears >= 8 {
			seniority = "Senior"
		} else if *candidate.ExperienceYears <= 2 {
			seniority = "Junior"
		}
	}
	summary := strPtr(candidate.ResumeSummary)
	if summary == "" && match != nil {
		summary = match.WhyShortlisted
	}
	return map[string]any{
		"professional_summary": summary,
		"strengths":            matchStrengths(match),
		"weaknesses":           matchMissing(match),
		"career_growth":        fmt.Sprintf("Current designation %s; evaluate trajectory toward %s.", strPtr(candidate.CurrentDesignation), job.Title),
		"risk_factors":         matchMissing(match),
		"recommended_role":     job.Title,
		"seniority":            seniority,
		"notice_period":        "Unknown",
	}
}

func resumeInsights(candidate *domain.Candidate, match *domain.SemanticMatch) map[string]any {
	text := strPtr(candidate.ResumeText)
	length := len(strings.TrimSpace(text))
	quality := 55.0
	if length > 800 {
		quality += 15
	}
	if candidate.ResumeSummary != nil && strings.TrimSpace(*candidate.ResumeSummary) != "" {
		quality += 10
	}
	if len(candidate.Skills) >= 5 {
		quality += 10
	}
	if quality > 100 {
		quality = 100
	}
	missingSections := []string{}
	if !strings.Contains(strings.ToLower(text), "experience") {
		missingSections = append(missingSections, "Experience section unclear")
	}
	if !strings.Contains(strings.ToLower(text), "education") {
		missingSections = append(missingSections, "Education")
	}
	if !strings.Contains(strings.ToLower(text), "project") {
		missingSections = append(missingSections, "Projects")
	}
	ats := quality
	if len(matchMissing(match)) > 3 {
		ats -= 10
	}
	return map[string]any{
		"resume_quality":    quality,
		"ats_score":         ats,
		"grammar":           "Not fully analyzed — use LLM polish when available",
		"formatting":        map[string]any{"length_chars": length, "has_summary": candidate.ResumeSummary != nil},
		"missing_sections":  missingSections,
		"skill_suggestions": matchMissing(match),
	}
}

func formatSummaryAnswer(m map[string]any) string {
	return fmt.Sprintf(
		"Professional Summary: %v\nStrengths: %v\nWeaknesses: %v\nRecommended Role: %v\nSeniority: %v\nNotice Period: %v",
		m["professional_summary"], m["strengths"], m["weaknesses"], m["recommended_role"], m["seniority"], m["notice_period"],
	)
}

func formatInsightsAnswer(m map[string]any) string {
	return fmt.Sprintf(
		"Resume Quality: %.0f\nATS Score: %.0f\nMissing Sections: %v\nSkill Suggestions: %v",
		m["resume_quality"], m["ats_score"], m["missing_sections"], m["skill_suggestions"],
	)
}

func formatCompareAnswer(rows []map[string]any, winner uuid.UUID) string {
	b := strings.Builder{}
	b.WriteString("Candidate comparison (ranked by AI Match):\n")
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("- %v: AI Match %v%% · %v\n", row["candidate_name"], row["ai_match"], row["recommendation"]))
	}
	b.WriteString(fmt.Sprintf("\nWinner: %s\n", winner.String()))
	return b.String()
}

func matchScore(m *domain.SemanticMatch) float64 {
	if m == nil {
		return 0
	}
	if m.AIMatchScore > 0 {
		return m.AIMatchScore
	}
	return m.SimilarityScore
}

func matchConfidence(m *domain.SemanticMatch) string {
	if m == nil || m.Confidence == "" {
		return "low"
	}
	return m.Confidence
}

func matchStrengths(m *domain.SemanticMatch) []string {
	if m == nil || m.Strengths == nil {
		return []string{}
	}
	return m.Strengths
}

func matchMissing(m *domain.SemanticMatch) []string {
	if m == nil || m.MissingSkills == nil {
		return []string{}
	}
	return m.MissingSkills
}

func breakdownVal(m domain.SemanticMatch, key string) float64 {
	if m.AIMatchBreakdown == nil {
		return 0
	}
	switch key {
	case "skills":
		return m.AIMatchBreakdown.Skills
	case "experience":
		return m.AIMatchBreakdown.Experience
	case "projects":
		return m.AIMatchBreakdown.Projects
	case "education":
		return m.AIMatchBreakdown.Education
	default:
		return 0
	}
}

func strPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func intPtr(v *int) any {
	if v == nil {
		return "unknown"
	}
	return *v
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}