package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/utils"

	"github.com/google/uuid"
)

var (
	ErrCandidateNotFound      = errors.New("candidate not found")
	ErrCandidateEmailExists   = errors.New("candidate email already exists")
	ErrInvalidCandidateStatus = errors.New("invalid candidate status")
	ErrInvalidJobReference    = errors.New("invalid job reference")
	ErrInvalidStatusTransition = errors.New("invalid candidate status transition")
)

var validCandidateStatuses = map[string]struct{}{
	domain.CandidateStatusApplied:              {},
	domain.CandidateStatusScreening:            {},
	domain.CandidateStatusShortlisted:          {},
	domain.CandidateStatusRecruiterShortlisted: {},
	domain.CandidateStatusAIShortlisted:        {},
	domain.CandidateStatusInterview:            {},
	domain.CandidateStatusSelected:             {},
	domain.CandidateStatusOffer:                {},
	domain.CandidateStatusHired:                {},
	domain.CandidateStatusRejected:             {},
}

var validProcessingStatuses = map[string]struct{}{
	domain.ProcessingStatusPending:    {},
	domain.ProcessingStatusProcessing: {},
	domain.ProcessingStatusCompleted:  {},
	domain.ProcessingStatusFailed:     {},
}

// allowedStatusTransitions keeps the recruiter pipeline consistent.
var allowedStatusTransitions = map[string]map[string]struct{}{
	domain.CandidateStatusApplied: {
		domain.CandidateStatusScreening:            {},
		domain.CandidateStatusShortlisted:          {},
		domain.CandidateStatusAIShortlisted:        {},
		domain.CandidateStatusRecruiterShortlisted: {},
		domain.CandidateStatusInterview:            {},
		domain.CandidateStatusRejected:             {},
	},
	domain.CandidateStatusScreening: {
		domain.CandidateStatusShortlisted:          {},
		domain.CandidateStatusAIShortlisted:        {},
		domain.CandidateStatusRecruiterShortlisted: {},
		domain.CandidateStatusInterview:            {},
		domain.CandidateStatusRejected:             {},
		domain.CandidateStatusApplied:              {},
	},
	domain.CandidateStatusShortlisted: {
		domain.CandidateStatusAIShortlisted:        {},
		domain.CandidateStatusRecruiterShortlisted: {},
		domain.CandidateStatusInterview:            {},
		domain.CandidateStatusSelected:             {},
		domain.CandidateStatusRejected:             {},
	},
	domain.CandidateStatusAIShortlisted: {
		domain.CandidateStatusRecruiterShortlisted: {},
		domain.CandidateStatusInterview:            {},
		domain.CandidateStatusSelected:             {},
		domain.CandidateStatusRejected:             {},
		domain.CandidateStatusShortlisted:          {},
	},
	domain.CandidateStatusRecruiterShortlisted: {
		domain.CandidateStatusInterview: {},
		domain.CandidateStatusSelected:  {},
		domain.CandidateStatusRejected:  {},
		domain.CandidateStatusOffer:     {},
	},
	domain.CandidateStatusInterview: {
		domain.CandidateStatusSelected:             {},
		domain.CandidateStatusRejected:             {},
		domain.CandidateStatusOffer:                {},
		domain.CandidateStatusRecruiterShortlisted: {},
	},
	domain.CandidateStatusSelected: {
		domain.CandidateStatusOffer:    {},
		domain.CandidateStatusHired:    {},
		domain.CandidateStatusRejected: {},
	},
	domain.CandidateStatusOffer: {
		domain.CandidateStatusHired:    {},
		domain.CandidateStatusRejected: {},
		domain.CandidateStatusSelected: {},
	},
	domain.CandidateStatusHired: {
		domain.CandidateStatusRejected: {},
	},
	domain.CandidateStatusRejected: {
		domain.CandidateStatusApplied:              {},
		domain.CandidateStatusScreening:            {},
		domain.CandidateStatusAIShortlisted:        {},
		domain.CandidateStatusRecruiterShortlisted: {},
		domain.CandidateStatusInterview:            {},
	},
}

func validateStatusTransition(from, to string) error {
	from = normalizeCandidateStatus(from)
	to = normalizeCandidateStatus(to)
	if from == "" {
		from = domain.CandidateStatusApplied
	}
	if to == "" {
		to = domain.CandidateStatusApplied
	}
	if from == to {
		return nil
	}
	if _, ok := validCandidateStatuses[to]; !ok {
		return ErrInvalidCandidateStatus
	}
	allowed, ok := allowedStatusTransitions[from]
	if !ok {
		return ErrInvalidStatusTransition
	}
	if _, ok := allowed[to]; !ok {
		return ErrInvalidStatusTransition
	}
	return nil
}

type CandidateInput struct {
	JobID              *uuid.UUID
	Name               string
	Email              string
	Phone              string
	ExperienceYears    *int
	CurrentCompany     string
	CurrentDesignation string
	Location           string
	Skills             []string
	Status             string
	ResumeURL          string
	ResumeText         string
	ResumeSummary      string
	Source             string
	ParsingStatus      string
	EmbeddingStatus    string
}

type CandidateService struct {
	candidates *repository.CandidateRepository
	jobs       *repository.JobRepository
	weights    config.FitScoreWeights
	embeddings *EmbeddingService
	resumes    *repository.ResumeRepository
	workspace  *CandidateWorkspaceService
	hooks      EnterpriseHooks
}

// EnterpriseHooks fans out in-app notifications and audit events (Phase 5).
type EnterpriseHooks interface {
	NotifyCompany(ctx context.Context, companyID uuid.UUID, typ, title, body string, entityType *string, entityID *uuid.UUID)
	Audit(ctx context.Context, companyID uuid.UUID, actor *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, meta map[string]any) error
}

func NewCandidateService(
	candidates *repository.CandidateRepository,
	jobs *repository.JobRepository,
	weights config.FitScoreWeights,
	embeddings *EmbeddingService,
	resumes *repository.ResumeRepository,
) *CandidateService {
	return &CandidateService{candidates: candidates, jobs: jobs, weights: weights, embeddings: embeddings, resumes: resumes}
}

func (s *CandidateService) SetWorkspace(workspace *CandidateWorkspaceService) {
	if s != nil {
		s.workspace = workspace
	}
}

func (s *CandidateService) SetEnterpriseHooks(hooks EnterpriseHooks) {
	if s != nil {
		s.hooks = hooks
	}
}

func (s *CandidateService) Create(ctx context.Context, companyID uuid.UUID, input CandidateInput) (*domain.Candidate, error) {
	if err := validateCandidateInput(input); err != nil {
		return nil, err
	}

	var job *domain.Job
	if input.JobID != nil {
		var err error
		job, err = s.loadJob(ctx, companyID, *input.JobID)
		if err != nil {
			return nil, err
		}
	}

	candidate := inputToCandidate(input)
	candidate.CompanyID = companyID
	s.applyScore(candidate, job)

	if err := s.candidates.Create(ctx, candidate); err != nil {
		if errors.Is(err, repository.ErrCandidateEmailExists) {
			return nil, ErrCandidateEmailExists
		}
		return nil, fmt.Errorf("create candidate: %w", err)
	}
	if strings.TrimSpace(input.ResumeText) != "" && s.resumes != nil {
		parsed := strings.TrimSpace(input.ResumeText)

		resume := &domain.Resume{
			ID:            uuid.New(),
			CandidateID:   &candidate.ID,
			CompanyID:     companyID,
			FileName:      "seed_resume.txt",
			FileURL:       "",
			StoragePath:   "",
			FileSize:      int64(len(parsed)),
			MimeType:      "text/plain",
			ParsedText:    &parsed,
			IsPrimary:     true,
			ParsingStatus: domain.ResumeParsingCompleted,
		}

		if err := s.resumes.CreateWithID(ctx, resume); err == nil && s.embeddings != nil {
			s.embeddings.EnqueueResume(companyID, resume.ID, parsed)
		}
	}

	if s.embeddings != nil {
		s.embeddings.EnqueueCandidate(candidate)
	}

	if s.workspace != nil {
		s.workspace.RecordApplied(ctx, companyID, candidate.ID)
	}

	return candidate, nil
}

func (s *CandidateService) GetByID(ctx context.Context, companyID, candidateID uuid.UUID) (*domain.Candidate, error) {
	candidate, err := s.candidates.GetByID(ctx, companyID, candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}
	if err := s.ensureCandidateScored(ctx, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (s *CandidateService) List(
	ctx context.Context,
	companyID uuid.UUID,
	page, limit int,
	status, search string,
	jobID *uuid.UUID,
	sort string,
) (*domain.CandidateListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 500 {
		limit = 500
	}

	normalizedStatus := normalizeCandidateStatus(status)
	if normalizedStatus != "" {
		if _, ok := validCandidateStatuses[normalizedStatus]; !ok {
			return nil, ErrInvalidCandidateStatus
		}
	}

	if jobID != nil {
		if err := s.ensureJobBelongsToCompany(ctx, companyID, *jobID); err != nil {
			return nil, err
		}
	}

	result, err := s.candidates.List(ctx, companyID, page, limit, repository.CandidateListFilter{
		Status: normalizedStatus,
		Search: strings.TrimSpace(search),
		JobID:  jobID,
		Sort:   sort,
	})
	if err != nil {
		return nil, err
	}

	for i := range result.Candidates {
		if err := s.ensureCandidateScored(ctx, &result.Candidates[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *CandidateService) Update(ctx context.Context, companyID, candidateID uuid.UUID, input CandidateInput) (*domain.Candidate, error) {
	if err := validateCandidateInput(input); err != nil {
		return nil, err
	}

	var job *domain.Job
	if input.JobID != nil {
		var err error
		job, err = s.loadJob(ctx, companyID, *input.JobID)
		if err != nil {
			return nil, err
		}
	}

	existing, err := s.candidates.GetByID(ctx, companyID, candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}

	updated := inputToCandidate(input)
	if err := validateStatusTransition(existing.Status, updated.Status); err != nil {
		return nil, err
	}
	updated.ID = existing.ID
	updated.CompanyID = existing.CompanyID
	updated.CreatedAt = existing.CreatedAt
	s.applyScore(updated, job)

	if err := s.candidates.Update(ctx, updated); err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		if errors.Is(err, repository.ErrCandidateEmailExists) {
			return nil, ErrCandidateEmailExists
		}
		return nil, err
	}

	if s.workspace != nil && existing.Status != updated.Status {
		s.workspace.RecordStatusChange(ctx, companyID, candidateID, existing.Status, updated.Status)
	}

	if s.hooks != nil && existing.Status != updated.Status {
		et := "candidate"
		cid := candidateID
		switch updated.Status {
		case domain.CandidateStatusRecruiterShortlisted, domain.CandidateStatusShortlisted, domain.CandidateStatusAIShortlisted:
			s.hooks.NotifyCompany(ctx, companyID, "candidate_shortlisted", "Candidate shortlisted",
				fmt.Sprintf("%s moved to %s", updated.Name, updated.Status), &et, &cid)
		case domain.CandidateStatusOffer:
			s.hooks.NotifyCompany(ctx, companyID, "offer_generated", "Offer stage",
				fmt.Sprintf("%s moved to offer", updated.Name), &et, &cid)
		}
		_ = s.hooks.Audit(ctx, companyID, nil, "candidate.status_change", "candidate", &cid, map[string]any{
			"from": existing.Status, "to": updated.Status,
		})
	}

	if s.embeddings != nil {
		s.embeddings.EnqueueCandidate(updated)
	}

	return updated, nil
}

func (s *CandidateService) Delete(ctx context.Context, companyID, candidateID uuid.UUID) error {
	err := s.candidates.Delete(ctx, companyID, candidateID)
	if errors.Is(err, repository.ErrCandidateNotFound) {
		return ErrCandidateNotFound
	}
	return err
}

// RescoreForJob recalculates fit scores for every candidate linked to the job.
func (s *CandidateService) RescoreForJob(ctx context.Context, companyID, jobID uuid.UUID) error {
	job, err := s.loadJob(ctx, companyID, jobID)
	if err != nil {
		return err
	}

	candidates, err := s.candidates.ListByJobID(ctx, companyID, jobID)
	if err != nil {
		return err
	}

	for i := range candidates {
		c := candidates[i]
		s.applyScore(&c, job)
		if err := s.candidates.Update(ctx, &c); err != nil {
			return fmt.Errorf("rescore candidate %s: %w", c.ID, err)
		}
	}
	return nil
}

// RescoreCandidate recomputes score for a single candidate (e.g. after resume attach).
func (s *CandidateService) RescoreCandidate(ctx context.Context, companyID, candidateID uuid.UUID) error {
	candidate, err := s.candidates.GetByID(ctx, companyID, candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return ErrCandidateNotFound
		}
		return err
	}
	return s.ensureCandidateScored(ctx, candidate)
}

// RescoreAllMissing backfills scores for candidates that have a job but no overall_score.
func (s *CandidateService) RescoreAllMissing(ctx context.Context) error {
	candidates, err := s.candidates.ListUnscoredWithJob(ctx)
	if err != nil {
		return err
	}
	for i := range candidates {
		if err := s.ensureCandidateScored(ctx, &candidates[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *CandidateService) ensureCandidateScored(ctx context.Context, candidate *domain.Candidate) error {
	if candidate == nil {
		return nil
	}
	if candidate.JobID == nil {
		if candidate.OverallScore != nil || candidate.LastScoredAt != nil {
			utils.ClearFitScore(candidate)
			return s.candidates.Update(ctx, candidate)
		}
		return nil
	}
	if candidate.OverallScore != nil && candidate.ScoreBreakdown != nil && candidate.LastScoredAt != nil {
		return nil
	}

	job, err := s.loadJob(ctx, candidate.CompanyID, *candidate.JobID)
	if err != nil {
		// Job may have been deleted (ON DELETE SET NULL should clear job_id); skip quietly.
		if errors.Is(err, ErrInvalidJobReference) {
			utils.ClearFitScore(candidate)
			candidate.JobID = nil
			_ = s.candidates.Update(ctx, candidate)
			return nil
		}
		return err
	}
	s.applyScore(candidate, job)
	return s.candidates.Update(ctx, candidate)
}

func (s *CandidateService) applyScore(candidate *domain.Candidate, job *domain.Job) {
	if job == nil {
		utils.ClearFitScore(candidate)
		return
	}
	result := utils.CompareCandidateToJob(candidate, job, s.weights)
	utils.ApplyFitScore(candidate, result)
}

func (s *CandidateService) loadJob(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrInvalidJobReference
		}
		return nil, err
	}
	if job == nil {
		return nil, ErrInvalidJobReference
	}
	return job, nil
}

func (s *CandidateService) ensureJobBelongsToCompany(ctx context.Context, companyID, jobID uuid.UUID) error {
	_, err := s.loadJob(ctx, companyID, jobID)
	return err
}

func validateCandidateInput(input CandidateInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(input.Email) == "" {
		return fmt.Errorf("email is required")
	}

	status := normalizeCandidateStatus(input.Status)
	if status == "" {
		status = domain.CandidateStatusApplied
	}
	if _, ok := validCandidateStatuses[status]; !ok {
		return ErrInvalidCandidateStatus
	}

	parsingStatus := normalizeProcessingStatus(input.ParsingStatus, domain.ProcessingStatusPending)
	if _, ok := validProcessingStatuses[parsingStatus]; !ok {
		return fmt.Errorf("invalid parsing status")
	}

	embeddingStatus := normalizeProcessingStatus(input.EmbeddingStatus, domain.ProcessingStatusPending)
	if _, ok := validProcessingStatuses[embeddingStatus]; !ok {
		return fmt.Errorf("invalid embedding status")
	}

	return nil
}

func inputToCandidate(input CandidateInput) *domain.Candidate {
	status := normalizeCandidateStatus(input.Status)
	if status == "" {
		status = domain.CandidateStatusApplied
	}

	skills := input.Skills
	if skills == nil {
		skills = []string{}
	}

	candidate := &domain.Candidate{
		JobID:           input.JobID,
		Name:            strings.TrimSpace(input.Name),
		Email:           strings.ToLower(strings.TrimSpace(input.Email)),
		Skills:          skills,
		Status:          status,
		ParsingStatus:   normalizeProcessingStatus(input.ParsingStatus, domain.ProcessingStatusPending),
		EmbeddingStatus: normalizeProcessingStatus(input.EmbeddingStatus, domain.ProcessingStatusPending),
		MatchedSkills:   []string{},
		MissingSkills:   []string{},
	}

	if v := strings.TrimSpace(input.Phone); v != "" {
		candidate.Phone = &v
	}
	if input.ExperienceYears != nil {
		candidate.ExperienceYears = input.ExperienceYears
	}
	if v := strings.TrimSpace(input.CurrentCompany); v != "" {
		candidate.CurrentCompany = &v
	}
	if v := strings.TrimSpace(input.CurrentDesignation); v != "" {
		candidate.CurrentDesignation = &v
	}
	if v := strings.TrimSpace(input.Location); v != "" {
		candidate.Location = &v
	}
	if v := strings.TrimSpace(input.ResumeURL); v != "" {
		candidate.ResumeURL = &v
	}
	if v := strings.TrimSpace(input.ResumeText); v != "" {
		candidate.ResumeText = &v
	}
	if v := strings.TrimSpace(input.ResumeSummary); v != "" {
		candidate.ResumeSummary = &v
	}
	if v := strings.TrimSpace(input.Source); v != "" {
		candidate.Source = &v
	}

	return candidate
}

func normalizeCandidateStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "applied":
		return domain.CandidateStatusApplied
	case "screening":
		return domain.CandidateStatusScreening
	case "shortlisted":
		return domain.CandidateStatusShortlisted
	case "recruiter_shortlisted":
		return domain.CandidateStatusRecruiterShortlisted
	case "ai_shortlisted":
		return domain.CandidateStatusAIShortlisted
	case "interview":
		return domain.CandidateStatusInterview
	case "selected":
		return domain.CandidateStatusSelected
	case "offer":
		return domain.CandidateStatusOffer
	case "hired":
		return domain.CandidateStatusHired
	case "rejected":
		return domain.CandidateStatusRejected
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func normalizeProcessingStatus(status, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	if value == "" {
		return fallback
	}
	return value
}
