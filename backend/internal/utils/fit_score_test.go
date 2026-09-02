package utils

import (
	"testing"
	"time"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
)

func testWeights() config.FitScoreWeights {
	return config.FitScoreWeights{
		Skills: 0.35, Experience: 0.20, Education: 0.15,
		Seniority: 0.15, Location: 0.10, Certifications: 0.05,
	}
}

func TestCompareCandidateToJobPerfectMatch(t *testing.T) {
	jobLoc := "Bangalore, India"
	expReq := "5 years"
	desc := "Looking for a Senior Software Engineer with a Bachelor's degree. AWS Certified Developer preferred."
	job := &domain.Job{
		Title:              "Senior Software Engineer",
		Location:           &jobLoc,
		ExperienceRequired: &expReq,
		Description:        &desc,
		RequiredSkills:     []string{"Go", "PostgreSQL", "Docker"},
	}
	years := 6
	title := "Senior Backend Engineer"
	candLoc := "Bangalore"
	resumeText := `---ATS_META_V1---
{"education":[{"degree":"Bachelor of Technology","branch":"Computer Science","school":"IIT"}],"certifications":["AWS Certified Developer"],"projects":[]}
---ATS_RAW---
body`
	candidate := &domain.Candidate{
		Name:               "Ada",
		Skills:             []string{"Golang", "Postgres", "Docker"},
		ExperienceYears:    &years,
		CurrentDesignation: &title,
		Location:           &candLoc,
		ResumeText:         &resumeText,
	}

	result := CompareCandidateToJob(candidate, job, testWeights())
	if result.OverallScore < 85 {
		t.Fatalf("expected high overall score, got %.2f breakdown=%+v matched=%v missing=%v",
			result.OverallScore, result.Breakdown, result.MatchedSkills, result.MissingSkills)
	}
	if len(result.MatchedSkills) != 3 {
		t.Fatalf("expected 3 matched skills, got %v", result.MatchedSkills)
	}
	if len(result.MissingSkills) != 0 {
		t.Fatalf("expected no missing skills, got %v", result.MissingSkills)
	}
	if result.Breakdown.Skills != 100 {
		t.Fatalf("skills score want 100 got %.2f", result.Breakdown.Skills)
	}
	if result.LastScoredAt.IsZero() {
		t.Fatal("last_scored_at should be set")
	}
}

func TestCompareCandidateToJobMissingSkillsAndExperience(t *testing.T) {
	expReq := "10 years"
	job := &domain.Job{
		Title:              "Staff Engineer",
		ExperienceRequired: &expReq,
		RequiredSkills:     []string{"Kubernetes", "Go", "AWS"},
	}
	years := 2
	title := "Junior Developer"
	candidate := &domain.Candidate{
		Skills:             []string{"Go"},
		ExperienceYears:    &years,
		CurrentDesignation: &title,
	}

	result := CompareCandidateToJob(candidate, job, testWeights())
	if len(result.MatchedSkills) != 1 {
		t.Fatalf("matched skills: %v", result.MatchedSkills)
	}
	if len(result.MissingSkills) != 2 {
		t.Fatalf("missing skills: %v", result.MissingSkills)
	}
	if result.Breakdown.Skills < 30 || result.Breakdown.Skills > 40 {
		t.Fatalf("skills ~33.33, got %.2f", result.Breakdown.Skills)
	}
	if result.Breakdown.Experience != 20 {
		t.Fatalf("experience want 20, got %.2f", result.Breakdown.Experience)
	}
	if result.OverallScore <= 0 || result.OverallScore >= 100 {
		t.Fatalf("overall should be mid-low, got %.2f", result.OverallScore)
	}
}

func TestCompareCandidateToJobNoJobRequirementsGivesFullPartial(t *testing.T) {
	job := &domain.Job{
		Title:          "Engineer",
		RequiredSkills: []string{},
	}
	years := 3
	candidate := &domain.Candidate{
		Skills:          []string{"Go"},
		ExperienceYears: &years,
	}
	result := CompareCandidateToJob(candidate, job, testWeights())
	if result.Breakdown.Skills != 100 {
		t.Fatalf("no required skills => skills 100, got %.2f", result.Breakdown.Skills)
	}
	if result.Breakdown.Experience != 100 {
		t.Fatalf("no experience req => 100, got %.2f", result.Breakdown.Experience)
	}
	if result.Breakdown.Location != 100 {
		t.Fatalf("no location req => 100, got %.2f", result.Breakdown.Location)
	}
}

func TestCompareCandidateToJobDeterministic(t *testing.T) {
	job := &domain.Job{
		Title:          "Senior Engineer",
		RequiredSkills: []string{"Python", "SQL"},
	}
	years := 4
	title := "Senior Engineer"
	candidate := &domain.Candidate{
		Skills:             []string{"Python", "SQL"},
		ExperienceYears:    &years,
		CurrentDesignation: &title,
	}
	a := CompareCandidateToJob(candidate, job, testWeights())
	time.Sleep(2 * time.Millisecond)
	b := CompareCandidateToJob(candidate, job, testWeights())
	if a.OverallScore != b.OverallScore || a.Breakdown != b.Breakdown {
		t.Fatalf("scores should be deterministic: %+v vs %+v", a, b)
	}
	if len(a.MatchedSkills) != len(b.MatchedSkills) {
		t.Fatal("matched skills should match")
	}
}

func TestExtractResumeMeta(t *testing.T) {
	text := `hello
---ATS_META_V1---
{"education":[{"degree":"MBA","branch":"","school":"X"}],"certifications":["PMP"],"projects":[]}
---ATS_RAW---
raw body`
	edu, certs := ExtractResumeMeta(&text)
	if len(edu) != 1 || edu[0].Degree != "MBA" {
		t.Fatalf("education: %+v", edu)
	}
	if len(certs) != 1 || certs[0] != "PMP" {
		t.Fatalf("certs: %v", certs)
	}
}

func TestApplyAndClearFitScore(t *testing.T) {
	c := &domain.Candidate{}
	ApplyFitScore(c, FitScoreResult{
		OverallScore:  88.5,
		Breakdown:     domain.FitScoreBreakdown{Skills: 100},
		MatchedSkills: []string{"Go"},
		MissingSkills: []string{"Rust"},
		LastScoredAt:  time.Now().UTC(),
	})
	if c.OverallScore == nil || *c.OverallScore != 88.5 {
		t.Fatal("overall not applied")
	}
	if c.ScoreBreakdown == nil || c.ScoreBreakdown.Skills != 100 {
		t.Fatal("breakdown not applied")
	}
	ClearFitScore(c)
	if c.OverallScore != nil || c.ScoreBreakdown != nil || c.LastScoredAt != nil {
		t.Fatal("expected cleared scores")
	}
}

func TestParseRequiredYears(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"5+ years", 5, true},
		{"3 yrs", 3, true},
		{"10", 10, true},
		{"", 0, false},
	}
	for _, tc := range cases {
		var ptr *string
		if tc.in != "" {
			s := tc.in
			ptr = &s
		}
		got := ParseRequiredYears(ptr)
		if tc.ok && (got == nil || *got != tc.want) {
			t.Fatalf("%q: want %.0f got %v", tc.in, tc.want, got)
		}
		if !tc.ok && got != nil {
			t.Fatalf("%q: want nil got %v", tc.in, got)
		}
	}
}
