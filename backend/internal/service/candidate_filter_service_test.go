package service

import (
	"context"
	"testing"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
)

func ptrString(s string) *string {
	return &s
}

func ptrInt(v int) *int {
	return &v
}

func TestRoleGateRejectsUnrelatedRoles(t *testing.T) {
	service := NewCandidateFilterService(nil, nil, DefaultCandidateFilterWeights(), 40)
	if service.roleMatches(ptrString("Security Engineer"), "Backend Engineer") {
		t.Fatalf("expected unrelated role to be rejected")
	}
}

func TestRoleGateAcceptsConfiguredSynonyms(t *testing.T) {
	service := NewCandidateFilterService(nil, nil, DefaultCandidateFilterWeights(), 40)
	cases := []struct {
		candidate string
		job       string
	}{
		{candidate: "Software Engineer", job: "Backend Engineer"},
		{candidate: "Go Developer", job: "Backend Engineer"},
		{candidate: "SDE Backend", job: "Backend Engineer"},
		{candidate: "Recruiter", job: "HR Executive"},
		{candidate: "Talent Acquisition", job: "HR Executive"},
		{candidate: "ML Engineer", job: "AI Engineer"},
	}

	for _, tc := range cases {
		if !service.roleMatches(ptrString(tc.candidate), tc.job) {
			t.Fatalf("expected role %q to match job %q", tc.candidate, tc.job)
		}
	}
}

func TestRoleGateMatchesGenericRoleFamilies(t *testing.T) {
	service := NewCandidateFilterService(nil, nil, DefaultCandidateFilterWeights(), 40)
	cases := []struct {
		candidate string
		job       string
	}{
		{candidate: "Digital Marketing Executive", job: "Marketing Executive"},
		{candidate: "Senior QA Engineer", job: "QA Engineer"},
		{candidate: "Automation Tester", job: "QA Engineer"},
		{candidate: "Frontend Developer", job: "Frontend Engineer"},
		{candidate: "React Developer", job: "Frontend Engineer"},
		{candidate: "Full Stack Developer", job: "Full Stack Engineer"},
		{candidate: "Business Development Manager", job: "Business Development"},
		{candidate: "SEO Specialist", job: "Marketing Executive"},
	}

	for _, tc := range cases {
		if !service.roleMatches(ptrString(tc.candidate), tc.job) {
			t.Fatalf("expected role %q to match job %q", tc.candidate, tc.job)
		}
	}
}

func TestScoreCandidateSkipsUnrelatedRoles(t *testing.T) {
	service := NewCandidateFilterService(nil, nil, DefaultCandidateFilterWeights(), 40)
	candidate := &domain.Candidate{
		CurrentDesignation: ptrString("Security Engineer"),
		Skills:             []string{"Go"},
		ExperienceYears:    ptrInt(3),
	}
	job := &domain.Job{
		Title:              "Backend Engineer",
		RequiredSkills:     []string{"Go"},
		ExperienceRequired: ptrString("2"),
	}

	if got := service.ScoreCandidate(candidate, job); got != 0 {
		t.Fatalf("expected unrelated role to score 0, got %.2f", got)
	}
}

func TestEligibleCandidateIDsForJobReturnsEmptyWhenNoApplicants(t *testing.T) {
	companyID := uuid.New()
	jobID := uuid.New()

	service := &CandidateFilterService{
		candidateRepo: &stubCandidateFilterRepo{
			listByJobIDFunc: func(ctx context.Context, companyIDArg, jobIDArg uuid.UUID) ([]domain.Candidate, error) {
				return []domain.Candidate{}, nil
			},
			listForSemanticFilteringFunc: func(ctx context.Context, companyIDArg uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error) {
				t.Fatalf("ListForSemanticFiltering must not be called when there are no applicants")
				return nil, nil
			},
		},
		jobRepo: &stubJobFilterRepo{
			getByIDFunc: func(ctx context.Context, companyIDArg, jobIDArg uuid.UUID) (*domain.Job, error) {
				return &domain.Job{ID: jobID, CompanyID: companyID, Title: "Backend Engineer"}, nil
			},
		},
		weights:   DefaultCandidateFilterWeights(),
		threshold: 40,
	}

	got, err := service.EligibleCandidateIDsForJob(context.Background(), companyID, jobID, nil)
	if err != nil {
		t.Fatalf("EligibleCandidateIDsForJob returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no eligible IDs for a job with zero applicants, got %v", got)
	}
}

func TestEligibleCandidateIDsForJobUsesJobApplicantsWhenInputEmpty(t *testing.T) {
	companyID := uuid.New()
	jobID := uuid.New()
	candidateA := uuid.New()
	candidateB := uuid.New()

	service := &CandidateFilterService{
		candidateRepo: &stubCandidateFilterRepo{
			listByJobIDFunc: func(ctx context.Context, companyIDArg, jobIDArg uuid.UUID) ([]domain.Candidate, error) {
				if companyIDArg != companyID || jobIDArg != jobID {
					t.Fatalf("unexpected company/job lookup %s/%s", companyIDArg, jobIDArg)
				}
				return []domain.Candidate{{ID: candidateA}, {ID: candidateB}}, nil
			},
			listForSemanticFilteringFunc: func(ctx context.Context, companyIDArg uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error) {
				if companyIDArg != companyID {
					t.Fatalf("unexpected company ID %s", companyIDArg)
				}
				if len(candidateIDs) != 2 || candidateIDs[0] != candidateA || candidateIDs[1] != candidateB {
					t.Fatalf("expected job applicants as filter input, got %v", candidateIDs)
				}
				return []domain.Candidate{
					{ID: candidateA, CurrentDesignation: ptrString("Backend Engineer"), Skills: []string{"Go"}, ExperienceYears: ptrInt(3)},
					{ID: candidateB, CurrentDesignation: ptrString("Backend Engineer"), Skills: []string{"Go"}, ExperienceYears: ptrInt(3)},
				}, nil
			},
		},
		jobRepo: &stubJobFilterRepo{
			getByIDFunc: func(ctx context.Context, companyIDArg, jobIDArg uuid.UUID) (*domain.Job, error) {
				return &domain.Job{ID: jobID, CompanyID: companyID, Title: "Backend Engineer"}, nil
			},
		},
		weights:   DefaultCandidateFilterWeights(),
		threshold: 40,
	}

	got, err := service.EligibleCandidateIDsForJob(context.Background(), companyID, jobID, nil)
	if err != nil {
		t.Fatalf("EligibleCandidateIDsForJob returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 job applicants to be returned, got %v", got)
	}
	if got[0] != candidateA || got[1] != candidateB {
		t.Fatalf("expected the job applicant IDs to be preserved, got %v", got)
	}
}

func TestRoleGateUsesResumeTextWhenDesignationMissing(t *testing.T) {
	service := NewCandidateFilterService(nil, nil, DefaultCandidateFilterWeights(), 40)
	candidate := &domain.Candidate{
		ResumeSummary:   ptrString("Full Stack Developer focused on Python and React APIs."),
		ResumeText:      ptrString("Professional summary\nFull Stack Developer with Python and React experience."),
		Skills:          []string{"Go", "React"},
		ExperienceYears: ptrInt(3),
	}
	job := &domain.Job{
		Title:              "Backend Engineer",
		RequiredSkills:     []string{"Go"},
		ExperienceRequired: ptrString("2"),
	}

	if got := service.ScoreCandidate(candidate, job); got <= 0 {
		t.Fatalf("expected resume-based role hint to allow scoring, got %.2f", got)
	}
}

type stubCandidateFilterRepo struct {
	listByJobIDFunc              func(ctx context.Context, companyID, jobID uuid.UUID) ([]domain.Candidate, error)
	listForSemanticFilteringFunc func(ctx context.Context, companyID uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error)
}

func (s *stubCandidateFilterRepo) ListByJobID(ctx context.Context, companyID, jobID uuid.UUID) ([]domain.Candidate, error) {
	if s.listByJobIDFunc != nil {
		return s.listByJobIDFunc(ctx, companyID, jobID)
	}
	return nil, nil
}

func (s *stubCandidateFilterRepo) ListForSemanticFiltering(ctx context.Context, companyID uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error) {
	if s.listForSemanticFilteringFunc != nil {
		return s.listForSemanticFilteringFunc(ctx, companyID, candidateIDs)
	}
	return nil, nil
}

type stubJobFilterRepo struct {
	getByIDFunc func(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error)
}

func (s *stubJobFilterRepo) GetByID(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(ctx, companyID, jobID)
	}
	return nil, nil
}
