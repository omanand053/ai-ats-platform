package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
)

// AIMatchResult is the recruiter-facing Overall AI Match for one job applicant.
type AIMatchResult struct {
	AIMatchScore         float64
	Confidence           string
	Strengths            []string
	MissingSkills        []string
	MatchedSkills        []string
	WhyShortlisted       string
	EligibilityScore     float64
	EligibilityBreakdown domain.FitScoreBreakdown
	Breakdown            domain.AIMatchBreakdown
}

// ComputeAIMatch combines semantic similarity with skills/experience/education/projects.
// Rule-based CompareCandidateToJob contributes to the score and eligibility details;
// it never blocks ranking.
func ComputeAIMatch(
	candidate *domain.Candidate,
	job *domain.Job,
	semanticScore float64,
	aiWeights config.AIMatchWeights,
	fitWeights config.FitScoreWeights,
) AIMatchResult {
	aiWeights = normalizeAIWeights(aiWeights)
	fit := CompareCandidateToJob(candidate, job, fitWeights)
	projects := ExtractResumeProjects(candidate.ResumeText)

	skillsScore := fit.Breakdown.Skills
	experienceScore := fit.Breakdown.Experience
	educationScore := fit.Breakdown.Education
	projectsScore := scoreProjects(projects, job.RequiredSkills)
	semantic := clamp(semanticScore, 0, 100)

	breakdown := domain.AIMatchBreakdown{
		Semantic:   round2(semantic),
		Skills:     round2(skillsScore),
		Experience: round2(experienceScore),
		Education:  round2(educationScore),
		Projects:   round2(projectsScore),
	}

	// Eligibility dimensions (skills/exp/edu/projects) soft-influence Overall AI Match.
	overall := semantic*aiWeights.Semantic +
		skillsScore*aiWeights.Skills +
		experienceScore*aiWeights.Experience +
		educationScore*aiWeights.Education +
		projectsScore*aiWeights.Projects

	matched := fit.MatchedSkills
	if matched == nil {
		matched = []string{}
	}
	missing := fit.MissingSkills
	if missing == nil {
		missing = []string{}
	}

	strengths := buildStrengths(matched, semantic, experienceScore, educationScore, projectsScore, candidate)
	why := buildWhyShortlisted(overall, semantic, matched, experienceScore, projectsScore)

	return AIMatchResult{
		AIMatchScore:         round2(clamp(overall, 0, 100)),
		Confidence:           confidenceBand(overall, semantic, len(matched), len(missing)),
		Strengths:            strengths,
		MissingSkills:        missing,
		MatchedSkills:        matched,
		WhyShortlisted:       why,
		EligibilityScore:     fit.OverallScore,
		EligibilityBreakdown: fit.Breakdown,
		Breakdown:            breakdown,
	}
}

func normalizeAIWeights(w config.AIMatchWeights) config.AIMatchWeights {
	sum := w.Semantic + w.Skills + w.Experience + w.Education + w.Projects
	if sum <= 0 {
		return config.AIMatchWeights{
			Semantic: 0.40, Skills: 0.25, Experience: 0.15,
			Education: 0.10, Projects: 0.10,
		}
	}
	w.Semantic /= sum
	w.Skills /= sum
	w.Experience /= sum
	w.Education /= sum
	w.Projects /= sum
	return w
}

func scoreProjects(projects []domain.ParsedProject, requiredSkills []string) float64 {
	if len(requiredSkills) == 0 {
		if len(projects) == 0 {
			return 50
		}
		return 100
	}
	if len(projects) == 0 {
		return 0
	}
	techKeys := make(map[string]struct{})
	for _, p := range projects {
		for _, t := range p.Technologies {
			if key := NormalizeSkillKey(t); key != "" {
				techKeys[key] = struct{}{}
			}
		}
		// Also scan description for required skill mentions.
		desc := strings.ToLower(p.Description + " " + p.Name)
		for _, req := range requiredSkills {
			key := NormalizeSkillKey(req)
			if key != "" && strings.Contains(desc, key) {
				techKeys[key] = struct{}{}
			}
		}
	}
	matched := 0
	for _, req := range requiredSkills {
		key := NormalizeSkillKey(req)
		if _, ok := techKeys[key]; ok {
			matched++
		}
	}
	return clamp(float64(matched)/float64(len(requiredSkills))*100, 0, 100)
}

func confidenceBand(aiMatch, semantic float64, matchedCount, missingCount int) string {
	if aiMatch >= 75 && semantic >= 60 && missingCount <= 2 {
		return "high"
	}
	if aiMatch >= 55 || (semantic >= 50 && matchedCount > 0) {
		return "medium"
	}
	return "low"
}

func buildStrengths(
	matchedSkills []string,
	semantic, experience, education, projects float64,
	candidate *domain.Candidate,
) []string {
	out := make([]string, 0, 6)
	if len(matchedSkills) > 0 {
		limit := len(matchedSkills)
		if limit > 5 {
			limit = 5
		}
		out = append(out, fmt.Sprintf("Strong skill overlap: %s", strings.Join(matchedSkills[:limit], ", ")))
	}
	if semantic >= 70 {
		out = append(out, "High resume↔job semantic similarity")
	} else if semantic >= 55 {
		out = append(out, "Solid resume↔job semantic similarity")
	}
	if experience >= 80 {
		out = append(out, "Experience meets or exceeds the role requirement")
	}
	if education >= 70 {
		out = append(out, "Education aligns with the role")
	}
	if projects >= 60 {
		out = append(out, "Relevant project technologies for this role")
	}
	if candidate != nil && candidate.CurrentDesignation != nil && strings.TrimSpace(*candidate.CurrentDesignation) != "" {
		out = append(out, fmt.Sprintf("Current role: %s", strings.TrimSpace(*candidate.CurrentDesignation)))
	}
	if len(out) == 0 {
		out = append(out, "Passed eligibility pre-filter for this job")
	}
	return out
}

func buildWhyShortlisted(aiMatch, semantic float64, matched []string, experience, projects float64) string {
	parts := make([]string, 0, 4)
	parts = append(parts, fmt.Sprintf("Overall AI Match %.0f%%", aiMatch))
	parts = append(parts, fmt.Sprintf("semantic similarity %.0f%%", semantic))
	if len(matched) > 0 {
		parts = append(parts, fmt.Sprintf("%d required skill(s) matched", len(matched)))
	}
	if experience >= 70 {
		parts = append(parts, "experience aligned")
	}
	if projects >= 60 {
		parts = append(parts, "relevant projects")
	}
	return strings.Join(parts, "; ") + "."
}

// ExtractResumeProjects pulls projects from ATS meta in resume_text.
func ExtractResumeProjects(resumeText *string) []domain.ParsedProject {
	if resumeText == nil || strings.TrimSpace(*resumeText) == "" {
		return nil
	}
	text := *resumeText
	start := strings.Index(text, metaStart)
	end := strings.Index(text, metaEnd)
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	raw := strings.TrimSpace(text[start+len(metaStart) : end])
	var meta resumeMetaPayload
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return meta.Projects
}
