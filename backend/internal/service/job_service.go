package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidJobStatus  = errors.New("invalid job status")
	ErrInvalidEmployment = errors.New("invalid employment type")
)

var validStatuses = map[string]struct{}{
	domain.JobStatusDraft:  {},
	domain.JobStatusOpen:   {},
	domain.JobStatusClosed: {},
}

var validEmploymentTypes = map[string]struct{}{
	domain.EmploymentFullTime:   {},
	domain.EmploymentPartTime:   {},
	domain.EmploymentContract:   {},
	domain.EmploymentInternship: {},
	domain.EmploymentTemporary:  {},
}

type JobInput struct {
	Title              string
	Department         string
	Location           string
	EmploymentType     string
	ExperienceRequired string
	Description        string
	RequiredSkills     []string
	Status             string
}

type JobService struct {
	jobs        *repository.JobRepository
	candidates  *CandidateService
	embeddings  *EmbeddingService
	defaultTopK int
}

func NewJobService(
	jobs *repository.JobRepository,
	candidates *CandidateService,
	embeddings *EmbeddingService,
	defaultTopK int,
) *JobService {
	if defaultTopK < 1 {
		defaultTopK = 200
	}
	service := &JobService{
		jobs:        jobs,
		candidates:  candidates,
		embeddings:  embeddings,
		defaultTopK: defaultTopK,
	}
	if service.embeddings != nil && service.candidates != nil {
		cfg, err := config.Load()
		if err == nil {
			service.embeddings.SetCandidateFilter(NewCandidateFilterService(
				service.candidates.candidates,
				service.jobs,
				CandidateFilterWeights{
					Role:       cfg.CandidateFilter.RoleWeight,
					Skills:     cfg.CandidateFilter.SkillsWeight,
					Experience: cfg.CandidateFilter.ExperienceWeight,
				},
				cfg.CandidateFilter.MinScore,
			))
			service.embeddings.SetMatchRepos(
				service.candidates.candidates,
				service.jobs,
				cfg.AIMatchWeights,
				cfg.FitScoreWeights,
			)
		} else {
			service.embeddings.SetCandidateFilter(NewCandidateFilterService(
				service.candidates.candidates,
				service.jobs,
				DefaultCandidateFilterWeights(),
				40,
			))
		}
	}
	return service
}

func (s *JobService) Create(ctx context.Context, companyID, userID uuid.UUID, input JobInput) (*domain.Job, error) {
	if err := validateJobInput(input); err != nil {
		return nil, err
	}

	job := inputToJob(input)
	job.CompanyID = companyID
	job.CreatedBy = &userID

	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	if s.embeddings != nil {
		s.embeddings.EnqueueJob(job)
	}

	return job, nil
}

func (s *JobService) GetByID(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return job, nil
}

func (s *JobService) List(ctx context.Context, companyID uuid.UUID, page, limit int, status string) (*domain.JobListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	normalizedStatus := normalizeStatus(status)
	if normalizedStatus != "" {
		if _, ok := validStatuses[normalizedStatus]; !ok {
			return nil, ErrInvalidJobStatus
		}
	}

	return s.jobs.List(ctx, companyID, page, limit, normalizedStatus)
}

func (s *JobService) Update(ctx context.Context, companyID, jobID uuid.UUID, input JobInput) (*domain.Job, error) {
	if err := validateJobInput(input); err != nil {
		return nil, err
	}

	existing, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	updated := inputToJob(input)
	updated.ID = existing.ID
	updated.CompanyID = existing.CompanyID
	updated.CreatedBy = existing.CreatedBy
	updated.CreatedAt = existing.CreatedAt

	if err := s.jobs.Update(ctx, updated); err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	if s.candidates != nil {
		if err := s.candidates.RescoreForJob(ctx, companyID, jobID); err != nil {
			log.Printf("warning: rescore candidates after job update %s: %v", jobID, err)
		}
	}

	if s.embeddings != nil {
		s.embeddings.EnqueueJob(updated)
	}

	return updated, nil
}

func (s *JobService) Delete(ctx context.Context, companyID, jobID uuid.UUID) error {
	err := s.jobs.Delete(ctx, companyID, jobID)
	if errors.Is(err, repository.ErrJobNotFound) {
		return ErrJobNotFound
	}
	return err
}

// SemanticMatches returns top-K candidates by resume↔job vector similarity.
func (s *JobService) SemanticMatches(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	topK int,
) (*domain.SemanticMatchResult, error) {
	if _, err := s.GetByID(ctx, companyID, jobID); err != nil {
		return nil, err
	}
	if s.embeddings == nil {
		return &domain.SemanticMatchResult{
			Status:  domain.SemanticStatusJobEmbeddingMissing,
			Message: "Embedding service is not configured.",
			JobID:   jobID,
			TopK:    topK,
			Matches: []domain.SemanticMatch{},
		}, nil
	}
	if topK < 1 {
		topK = s.defaultTopK
	}
	return s.embeddings.SemanticMatchesForJob(ctx, companyID, jobID, topK)
}

func validateJobInput(input JobInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("title is required")
	}

	employmentType := normalizeEmploymentType(input.EmploymentType)
	if employmentType == "" {
		employmentType = domain.EmploymentFullTime
	}
	if _, ok := validEmploymentTypes[employmentType]; !ok {
		return ErrInvalidEmployment
	}

	status := normalizeStatus(input.Status)
	if status == "" {
		status = domain.JobStatusDraft
	}
	if _, ok := validStatuses[status]; !ok {
		return ErrInvalidJobStatus
	}

	return nil
}

func inputToJob(input JobInput) *domain.Job {
	employmentType := normalizeEmploymentType(input.EmploymentType)
	if employmentType == "" {
		employmentType = domain.EmploymentFullTime
	}

	status := normalizeStatus(input.Status)
	if status == "" {
		status = domain.JobStatusDraft
	}

	skills := input.RequiredSkills
	if skills == nil {
		skills = []string{}
	}

	job := &domain.Job{
		Title:           strings.TrimSpace(input.Title),
		EmploymentType:  employmentType,
		RequiredSkills:  skills,
		Status:          status,
		EmbeddingStatus: domain.ProcessingStatusPending,
	}

	if v := strings.TrimSpace(input.Department); v != "" {
		job.Department = &v
	}
	if v := strings.TrimSpace(input.Location); v != "" {
		job.Location = &v
	}
	if v := strings.TrimSpace(input.ExperienceRequired); v != "" {
		job.ExperienceRequired = &v
	}
	if v := strings.TrimSpace(input.Description); v != "" {
		job.Description = &v
	}

	return job
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft":
		return domain.JobStatusDraft
	case "open":
		return domain.JobStatusOpen
	case "closed":
		return domain.JobStatusClosed
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func normalizeEmploymentType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
