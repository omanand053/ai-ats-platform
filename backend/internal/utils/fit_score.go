package utils

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
)

const (
	metaStart = "---ATS_META_V1---"
	metaEnd   = "---ATS_RAW---"
)

type resumeMetaPayload struct {
	Education      []domain.ParsedEducation `json:"education"`
	Certifications []string                 `json:"certifications"`
	Projects       []domain.ParsedProject   `json:"projects"`
}

// FitScoreResult is the deterministic output of CompareCandidateToJob.
type FitScoreResult struct {
	OverallScore   float64
	Breakdown      domain.FitScoreBreakdown
	MatchedSkills  []string
	MissingSkills  []string
	LastScoredAt   time.Time
}

// CompareCandidateToJob computes a weighted fit score (0–100).
func CompareCandidateToJob(candidate *domain.Candidate, job *domain.Job, weights config.FitScoreWeights) FitScoreResult {
	now := time.Now().UTC()
	if candidate == nil || job == nil {
		return FitScoreResult{
			MatchedSkills: []string{},
			MissingSkills: []string{},
			LastScoredAt:  now,
		}
	}

	education, certifications := ExtractResumeMeta(candidate.ResumeText)

	matched, missing := matchSkills(candidate.Skills, job.RequiredSkills)
	skillsScore := scoreSkills(matched, job.RequiredSkills)
	experienceScore := scoreExperience(candidate.ExperienceYears, job.ExperienceRequired)
	educationScore := scoreEducation(education, job)
	seniorityScore := scoreSeniority(candidate.CurrentDesignation, job.Title)
	locationScore := scoreLocation(candidate.Location, job.Location)
	certsScore := scoreCertifications(certifications, job)

	breakdown := domain.FitScoreBreakdown{
		Skills:         round2(skillsScore),
		Experience:     round2(experienceScore),
		Education:      round2(educationScore),
		Seniority:      round2(seniorityScore),
		Location:       round2(locationScore),
		Certifications: round2(certsScore),
	}

	overall := breakdown.Skills*weights.Skills +
		breakdown.Experience*weights.Experience +
		breakdown.Education*weights.Education +
		breakdown.Seniority*weights.Seniority +
		breakdown.Location*weights.Location +
		breakdown.Certifications*weights.Certifications

	if matched == nil {
		matched = []string{}
	}
	if missing == nil {
		missing = []string{}
	}

	return FitScoreResult{
		OverallScore:  round2(clamp(overall, 0, 100)),
		Breakdown:     breakdown,
		MatchedSkills: matched,
		MissingSkills: missing,
		LastScoredAt:  now,
	}
}

// ApplyFitScore writes scoring fields onto the candidate.
func ApplyFitScore(candidate *domain.Candidate, result FitScoreResult) {
	if candidate == nil {
		return
	}
	score := result.OverallScore
	candidate.OverallScore = &score
	bd := result.Breakdown
	candidate.ScoreBreakdown = &bd
	candidate.MatchedSkills = result.MatchedSkills
	candidate.MissingSkills = result.MissingSkills
	ts := result.LastScoredAt
	candidate.LastScoredAt = &ts
}

// ClearFitScore removes scoring fields (e.g. when job_id is cleared).
func ClearFitScore(candidate *domain.Candidate) {
	if candidate == nil {
		return
	}
	candidate.OverallScore = nil
	candidate.ScoreBreakdown = nil
	candidate.MatchedSkills = []string{}
	candidate.MissingSkills = []string{}
	candidate.LastScoredAt = nil
}

// ExtractResumeMeta pulls education and certifications from ATS meta in resume_text.
func ExtractResumeMeta(resumeText *string) ([]domain.ParsedEducation, []string) {
	if resumeText == nil || strings.TrimSpace(*resumeText) == "" {
		return nil, nil
	}
	text := *resumeText
	start := strings.Index(text, metaStart)
	end := strings.Index(text, metaEnd)
	if start < 0 || end < 0 || end <= start {
		return nil, nil
	}
	raw := strings.TrimSpace(text[start+len(metaStart) : end])
	var meta resumeMetaPayload
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, nil
	}
	certs := make([]string, 0, len(meta.Certifications))
	for _, c := range meta.Certifications {
		if v := strings.TrimSpace(c); v != "" {
			certs = append(certs, v)
		}
	}
	return meta.Education, certs
}

func matchSkills(candidateSkills, requiredSkills []string) (matched, missing []string) {
	matched = make([]string, 0)
	missing = make([]string, 0)
	candKeys := make(map[string]string, len(candidateSkills))
	for _, s := range candidateSkills {
		key := NormalizeSkillKey(s)
		if key == "" {
			continue
		}
		if _, ok := candKeys[key]; !ok {
			candKeys[key] = CanonicalSkillName(s)
		}
	}
	seenMatch := make(map[string]struct{})
	seenMissing := make(map[string]struct{})
	for _, req := range requiredSkills {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		key := NormalizeSkillKey(req)
		if name, ok := candKeys[key]; ok {
			if _, seen := seenMatch[key]; !seen {
				matched = append(matched, name)
				seenMatch[key] = struct{}{}
			}
		} else {
			if _, seen := seenMissing[key]; !seen {
				missing = append(missing, CanonicalSkillName(req))
				seenMissing[key] = struct{}{}
			}
		}
	}
	return matched, missing
}

func scoreSkills(matched, required []string) float64 {
	reqCount := 0
	for _, r := range required {
		if strings.TrimSpace(r) != "" {
			reqCount++
		}
	}
	if reqCount == 0 {
		return 100
	}
	return clamp(float64(len(matched))/float64(reqCount)*100, 0, 100)
}

var yearsPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*\+?\s*(?:years?|yrs?|y)?`)

func ParseRequiredYears(experienceRequired *string) *float64 {
	if experienceRequired == nil {
		return nil
	}
	s := strings.TrimSpace(*experienceRequired)
	if s == "" {
		return nil
	}
	m := yearsPattern.FindStringSubmatch(s)
	if m == nil {
		// bare number
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return &v
		}
		return nil
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	return &v
}

func scoreExperience(candidateYears *int, experienceRequired *string) float64 {
	required := ParseRequiredYears(experienceRequired)
	if required == nil || *required <= 0 {
		return 100
	}
	if candidateYears == nil {
		return 0
	}
	ratio := float64(*candidateYears) / *required
	return clamp(ratio*100, 0, 100)
}

func educationLevel(text string) int {
	s := strings.ToLower(text)
	switch {
	case strings.Contains(s, "phd") || strings.Contains(s, "doctorate") || strings.Contains(s, "doctoral"):
		return 4
	case strings.Contains(s, "master") || strings.Contains(s, "m.s") || strings.Contains(s, "m.tech") ||
		strings.Contains(s, "mba") || strings.Contains(s, "m.sc") || strings.Contains(s, "ms "):
		return 3
	case strings.Contains(s, "bachelor") || strings.Contains(s, "b.s") || strings.Contains(s, "b.tech") ||
		strings.Contains(s, "b.e") || strings.Contains(s, "b.sc") || strings.Contains(s, "undergraduate") ||
		strings.Contains(s, "degree"):
		return 2
	case strings.Contains(s, "associate") || strings.Contains(s, "diploma") || strings.Contains(s, "high school") ||
		strings.Contains(s, "secondary"):
		return 1
	default:
		return 0
	}
}

func candidateEducationLevel(education []domain.ParsedEducation) int {
	best := 0
	for _, e := range education {
		parts := []string{e.Degree, e.Branch, e.School}
		level := educationLevel(strings.Join(parts, " "))
		if level > best {
			best = level
		}
	}
	return best
}

func jobRequiredEducationLevel(job *domain.Job) int {
	parts := []string{}
	if job.Description != nil {
		parts = append(parts, *job.Description)
	}
	if job.Title != "" {
		parts = append(parts, job.Title)
	}
	text := strings.ToLower(strings.Join(parts, " "))
	// Prefer explicit degree requirements.
	patterns := []struct {
		re    *regexp.Regexp
		level int
	}{
		{regexp.MustCompile(`(?i)\b(ph\.?d|doctorate|doctoral)\b`), 4},
		{regexp.MustCompile(`(?i)\b(master'?s?|m\.?s\.?|m\.?tech|mba|m\.?sc)\b`), 3},
		{regexp.MustCompile(`(?i)\b(bachelor'?s?|b\.?s\.?|b\.?tech|b\.?e\.?|b\.?sc|undergraduate degree)\b`), 2},
		{regexp.MustCompile(`(?i)\b(associate'?s?|diploma|high school)\b`), 1},
	}
	for _, p := range patterns {
		if p.re.MatchString(text) {
			return p.level
		}
	}
	return 0
}

func scoreEducation(education []domain.ParsedEducation, job *domain.Job) float64 {
	required := jobRequiredEducationLevel(job)
	cand := candidateEducationLevel(education)
	if required == 0 {
		// No job education requirement: reward having any education listed.
		if cand > 0 {
			return 100
		}
		return 70
	}
	if cand == 0 {
		return 0
	}
	if cand >= required {
		return 100
	}
	// Partial credit for being one level below.
	gap := required - cand
	return clamp(100-float64(gap)*35, 0, 100)
}

var seniorityRanks = []struct {
	pattern *regexp.Regexp
	rank    int
}{
	{regexp.MustCompile(`(?i)\b(intern|trainee|apprentice)\b`), 1},
	{regexp.MustCompile(`(?i)\b(junior|jr\.?|entry[\s-]?level|associate)\b`), 2},
	{regexp.MustCompile(`(?i)\b(mid[\s-]?level|intermediate)\b`), 3},
	{regexp.MustCompile(`(?i)\b(senior|sr\.?)\b`), 4},
	{regexp.MustCompile(`(?i)\b(staff|principal)\b`), 5},
	{regexp.MustCompile(`(?i)\b(lead|tech[\s-]?lead)\b`), 5},
	{regexp.MustCompile(`(?i)\b(manager|director|head|vp|vice[\s-]?president|chief|cto|ceo)\b`), 6},
}

func seniorityRank(title string) int {
	s := strings.TrimSpace(title)
	if s == "" {
		return 0
	}
	best := 0
	for _, item := range seniorityRanks {
		if item.pattern.MatchString(s) && item.rank > best {
			best = item.rank
		}
	}
	if best == 0 {
		// Unlabeled IC title treated as mid-level.
		return 3
	}
	return best
}

func scoreSeniority(candidateTitle *string, jobTitle string) float64 {
	candTitle := ""
	if candidateTitle != nil {
		candTitle = *candidateTitle
	}
	jobRank := seniorityRank(jobTitle)
	candRank := seniorityRank(candTitle)
	if jobRank == 0 && candRank == 0 {
		return 70
	}
	if candTitle == "" {
		return 40
	}
	diff := math.Abs(float64(jobRank - candRank))
	switch {
	case diff == 0:
		return 100
	case diff == 1:
		return 80
	case diff == 2:
		return 55
	default:
		return 30
	}
}

func normalizeLocation(loc string) string {
	s := strings.ToLower(strings.TrimSpace(loc))
	s = strings.ReplaceAll(s, ",", " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func scoreLocation(candidateLoc, jobLoc *string) float64 {
	if jobLoc == nil || strings.TrimSpace(*jobLoc) == "" {
		return 100
	}
	job := normalizeLocation(*jobLoc)
	if candidateLoc == nil || strings.TrimSpace(*candidateLoc) == "" {
		return 0
	}
	cand := normalizeLocation(*candidateLoc)

	if cand == job {
		return 100
	}
	if strings.Contains(cand, job) || strings.Contains(job, cand) {
		return 90
	}

	jobParts := strings.Fields(job)
	candParts := strings.Fields(cand)
	candSet := make(map[string]struct{}, len(candParts))
	for _, p := range candParts {
		if len(p) < 2 {
			continue
		}
		candSet[p] = struct{}{}
	}
	hits := 0
	for _, p := range jobParts {
		if len(p) < 2 {
			continue
		}
		if _, ok := candSet[p]; ok {
			hits++
		}
	}
	if hits == 0 {
		// Remote / hybrid keywords
		if (strings.Contains(job, "remote") && strings.Contains(cand, "remote")) ||
			(strings.Contains(job, "hybrid") && strings.Contains(cand, "hybrid")) {
			return 85
		}
		return 0
	}
	return clamp(float64(hits)/float64(len(jobParts))*100, 0, 100)
}

var commonCertPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(aws\s*(certified|solutions|developer|sysops|cloud)[^\n,]{0,40})\b`),
	regexp.MustCompile(`(?i)\b(azure\s*(fund|administrator|developer|solutions)[^\n,]{0,40})\b`),
	regexp.MustCompile(`(?i)\b(google\s*cloud|gcp)[^\n,]{0,30}\b`),
	regexp.MustCompile(`(?i)\b(cissp|cism|cisa|ceh|comptia\s*\w+|pmp|scrum\s*master|csm|cka|ckad|cks)\b`),
	regexp.MustCompile(`(?i)\b(certified\s+[a-z0-9][a-z0-9\s\-]{2,40})\b`),
}

func extractJobCertRequirements(job *domain.Job) []string {
	parts := []string{}
	if job.Description != nil {
		parts = append(parts, *job.Description)
	}
	text := strings.Join(parts, "\n")
	if strings.TrimSpace(text) == "" {
		return nil
	}
	found := make([]string, 0)
	seen := make(map[string]struct{})
	for _, re := range commonCertPatterns {
		matches := re.FindAllString(text, -1)
		for _, m := range matches {
			key := NormalizeSkillKey(m)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			found = append(found, strings.TrimSpace(m))
		}
	}
	return found
}

func scoreCertifications(candidateCerts []string, job *domain.Job) float64 {
	required := extractJobCertRequirements(job)
	if len(required) == 0 {
		if len(candidateCerts) > 0 {
			return 100
		}
		return 80
	}
	if len(candidateCerts) == 0 {
		return 0
	}
	candKeys := make(map[string]struct{}, len(candidateCerts))
	for _, c := range candidateCerts {
		candKeys[NormalizeSkillKey(c)] = struct{}{}
		// also store looser tokens
		for _, tok := range strings.Fields(NormalizeSkillKey(c)) {
			if len(tok) >= 3 {
				candKeys[tok] = struct{}{}
			}
		}
	}
	hits := 0
	for _, req := range required {
		key := NormalizeSkillKey(req)
		matched := false
		if _, ok := candKeys[key]; ok {
			matched = true
		} else {
			for _, tok := range strings.Fields(key) {
				if len(tok) >= 3 {
					if _, ok := candKeys[tok]; ok {
						matched = true
						break
					}
				}
			}
		}
		if matched {
			hits++
		}
	}
	return clamp(float64(hits)/float64(len(required))*100, 0, 100)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
