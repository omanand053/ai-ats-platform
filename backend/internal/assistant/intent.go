package assistant

import (
	"strings"
	"unicode"
)

// DetectIntent returns ATS_DATA | RAG | GENERAL using deterministic heuristics only.
// No LLM classification — matches the fast pre-LangChain routing behaviour.
func DetectIntent(query string) Intent {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return IntentGeneral
	}
	if intent, ok := heuristicIntent(q); ok {
		return intent
	}
	return IntentGeneral
}

func heuristicIntent(q string) (Intent, bool) {
	ragTriggers := []string{
		"summarize", "summary of resume", "uploaded resume", "uploaded document",
		"explain candidate strength", "candidate strengths", "review jd", "review job description",
		"compare uploaded", "compare resume", "interview notes", "extract skills",
		"resume with jd", "matching candidates from resume", "cover letter",
		"review cover", "resume review", "analyze resume", "parse resume",
		"document", "based on the resume", "from the resume", "jd review",
		"review the uploaded", "generate interview",
	}
	for _, t := range ragTriggers {
		if strings.Contains(q, t) {
			return IntentRAG, true
		}
	}

	atsTriggers := []string{
		"show backend candidates", "show frontend", "top frontend", "top backend",
		"highest experience", "most applied", "candidates with", "candidates shortlisted",
		"shortlisted", "jobs created", "hiring funnel", "average ai score", "avg ai",
		"interview schedule", "scheduled interview", "candidate comparison", "compare candidates",
		"resume score", "ai score", "how many candidates", "how many applicants",
		"total candidates", "total applicants", "total jobs", "open jobs",
		"application stats", "dashboard", "analytics", "overview", "funnel",
		"list candidates", "list jobs", "show candidates", "show jobs", "show applicants",
		"find candidates", "find jobs", "search candidates", "search jobs",
		"hired", "offer", "rejected", "applied", "screening",
		"ai shortlisted", "recruiter shortlisted", "most experience",
		"highest score", "top score", "jobs this month", "created this month",
	}
	for _, t := range atsTriggers {
		if strings.Contains(q, t) {
			return IntentATSData, true
		}
	}

	tokens := tokenize(q)
	atsTokens := map[string]struct{}{
		"candidate": {}, "candidates": {}, "applicant": {}, "applicants": {},
		"job": {}, "jobs": {}, "opening": {}, "openings": {}, "position": {}, "positions": {},
		"shortlist": {}, "shortlisted": {}, "funnel": {}, "interview": {}, "interviews": {},
		"hired": {}, "offer": {}, "pipeline": {},
	}
	atsHits := 0
	for _, t := range tokens {
		if _, ok := atsTokens[t]; ok {
			atsHits++
		}
	}
	if atsHits >= 1 && containsAny(q, []string{
		"show", "list", "find", "search", "how many", "count", "total", "top", "highest", "average", "avg", "schedule",
	}) {
		return IntentATSData, true
	}

	generalTriggers := []string{
		"roadmap", "interview question", "star method", "star interview",
		"resume format", "career advice", "difference between", "best practices",
		"how to prepare", "what is", "explain react", "explain golang",
		"java vs", "go vs", "software engineering", "system design",
	}
	for _, t := range generalTriggers {
		if strings.Contains(q, t) {
			return IntentGeneral, true
		}
	}

	return "", false
}

func tokenize(prompt string) []string {
	fields := strings.FieldsFunc(prompt, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ToLower(strings.TrimSpace(f))
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func containsAny(prompt string, triggers []string) bool {
	for _, t := range triggers {
		if strings.Contains(prompt, t) {
			return true
		}
	}
	return false
}
