package rag

import (
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/domain"
)

// ContextBuilder turns retrieved resume docs into an LLM context string.
type ContextBuilder struct {
	MaxCharsPerResume int
}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{MaxCharsPerResume: 2500}
}

// Build formats only the retrieved resumes into a bounded context block.
func (b *ContextBuilder) Build(job *domain.Job, docs []RetrievedDoc) string {
	if b == nil {
		b = NewContextBuilder()
	}
	maxChars := b.MaxCharsPerResume
	if maxChars < 500 {
		maxChars = 500
	}

	var sb strings.Builder
	sb.WriteString("JOB\n")
	if job != nil {
		sb.WriteString(fmt.Sprintf("Title: %s\n", job.Title))
		if job.Location != nil {
			sb.WriteString(fmt.Sprintf("Location: %s\n", *job.Location))
		}
		if job.ExperienceRequired != nil {
			sb.WriteString(fmt.Sprintf("Experience required: %s\n", *job.ExperienceRequired))
		}
		if len(job.RequiredSkills) > 0 {
			sb.WriteString(fmt.Sprintf("Required skills: %s\n", strings.Join(job.RequiredSkills, ", ")))
		}
		if job.Description != nil {
			sb.WriteString(fmt.Sprintf("Description: %s\n", truncate(*job.Description, 1200)))
		}
	}
	sb.WriteString("\nRETRIEVED CANDIDATE RESUMES (use only this context)\n")

	if len(docs) == 0 {
		sb.WriteString("(none)\n")
		return sb.String()
	}

	for i, doc := range docs {
		m := doc.Match
		fit := "n/a"
		if m.OverallScore != nil {
			fit = fmt.Sprintf("%.0f", *m.OverallScore)
		}
		resumeLabel := ResumeLabel(m.CandidateName, doc.ResumeFileName)
		sb.WriteString(fmt.Sprintf(
			"\n--- Candidate %d ---\nName: %s\nResume: %s\nSemantic similarity: %.1f%%\nRule-based fit score: %s\nResume text:\n%s\n",
			i+1,
			m.CandidateName,
			resumeLabel,
			m.SimilarityScore,
			fit,
			truncate(doc.ResumeText, maxChars),
		))
	}
	return sb.String()
}

// ResumeLabel is a human-friendly resume identifier for recruiters.
func ResumeLabel(candidateName, fileName string) string {
	candidateName = strings.TrimSpace(candidateName)
	fileName = strings.TrimSpace(fileName)
	if fileName != "" && candidateName != "" {
		return fmt.Sprintf("%s (%s)", candidateName, fileName)
	}
	if fileName != "" {
		return fileName
	}
	if candidateName != "" {
		return "Resume - " + candidateName
	}
	return "Resume"
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
