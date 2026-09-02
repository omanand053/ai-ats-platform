package utils

import (
	"testing"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
)

func TestComputeAIMatchRanksSemanticAndSkills(t *testing.T) {
	years := 5
	designation := "Backend Engineer"
	candidate := &domain.Candidate{
		ID:                 uuid.New(),
		Name:               "Ada",
		Skills:             []string{"Go", "PostgreSQL", "Docker"},
		ExperienceYears:    &years,
		CurrentDesignation: &designation,
	}
	exp := "3+ years"
	job := &domain.Job{
		ID:                 uuid.New(),
		Title:              "Backend Engineer",
		RequiredSkills:     []string{"Go", "PostgreSQL", "Kubernetes"},
		ExperienceRequired: &exp,
	}

	aiWeights := config.AIMatchWeights{
		Semantic: 0.4, Skills: 0.25, Experience: 0.15, Education: 0.1, Projects: 0.1,
	}
	fitWeights := config.FitScoreWeights{
		Skills: 0.35, Experience: 0.2, Education: 0.15,
		Seniority: 0.15, Location: 0.1, Certifications: 0.05,
	}

	got := ComputeAIMatch(candidate, job, 80, aiWeights, fitWeights)
	if got.AIMatchScore <= 0 || got.AIMatchScore > 100 {
		t.Fatalf("unexpected ai match score %.2f", got.AIMatchScore)
	}
	if got.Confidence == "" {
		t.Fatal("expected confidence band")
	}
	if len(got.MissingSkills) == 0 {
		t.Fatal("expected Kubernetes as missing skill")
	}
	if got.WhyShortlisted == "" {
		t.Fatal("expected why shortlisted text")
	}
	if got.EligibilityScore <= 0 {
		t.Fatal("expected eligibility score from rule-based fit")
	}
}
