package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ai-ats-platform/backend/internal/domain"
	embpkg "ai-ats-platform/backend/internal/embedding"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

// EmbeddingJob is a unit of async embedding work.
type EmbeddingJob struct {
	CompanyID  uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	Text       string
}

// EmbeddingService generates and persists embeddings via a swappable Provider.
type EmbeddingService struct {
	repo     *repository.EmbeddingRepository
	provider embpkg.Provider
	queue    chan EmbeddingJob
	wg       sync.WaitGroup
	once     sync.Once
}

func NewEmbeddingService(repo *repository.EmbeddingRepository, provider embpkg.Provider) *EmbeddingService {
	return &EmbeddingService{
		repo:     repo,
		provider: provider,
		queue:    make(chan EmbeddingJob, 256),
	}
}

// StartWorkers processes embedding jobs asynchronously.
func (s *EmbeddingService) StartWorkers(ctx context.Context, workers int) {
	if s == nil {
		return
	}
	if workers < 1 {
		workers = 2
	}
	s.once.Do(func() {
		for i := 0; i < workers; i++ {
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case job, ok := <-s.queue:
						if !ok {
							return
						}
						if err := s.Process(context.Background(), job); err != nil {
							log.Printf("embedding job %s/%s failed: %v", job.EntityType, job.EntityID, err)
						}
					}
				}
			}()
		}
	})
}

// Enqueue schedules embedding generation without blocking the caller.
func (s *EmbeddingService) Enqueue(job EmbeddingJob) {
	if s == nil {
		return
	}
	text := strings.TrimSpace(job.Text)
	if text == "" || job.EntityID == uuid.Nil {
		return
	}
	job.Text = text

	select {
	case s.queue <- job:
	default:
		// Queue full: still async, do not block HTTP handlers.
		go func(j EmbeddingJob) {
			if err := s.Process(context.Background(), j); err != nil {
				log.Printf("embedding overflow job %s/%s failed: %v", j.EntityType, j.EntityID, err)
			}
		}(job)
	}
}

// EnqueueResume embeds parsed resume text after upload.
func (s *EmbeddingService) EnqueueResume(companyID, resumeID uuid.UUID, parsedText string) {
	s.Enqueue(EmbeddingJob{
		CompanyID:  companyID,
		EntityType: domain.EmbeddingEntityResume,
		EntityID:   resumeID,
		Text:       parsedText,
	})
}

// EnqueueCandidate embeds candidate profile / resume text after save.
func (s *EmbeddingService) EnqueueCandidate(candidate *domain.Candidate) {
	if candidate == nil {
		return
	}
	s.Enqueue(EmbeddingJob{
		CompanyID:  candidate.CompanyID,
		EntityType: domain.EmbeddingEntityCandidate,
		EntityID:   candidate.ID,
		Text:       BuildCandidateEmbeddingText(candidate),
	})
}

// EnqueueJob embeds job description content after create/update.
func (s *EmbeddingService) EnqueueJob(job *domain.Job) {
	if job == nil {
		return
	}
	s.Enqueue(EmbeddingJob{
		CompanyID:  job.CompanyID,
		EntityType: domain.EmbeddingEntityJob,
		EntityID:   job.ID,
		Text:       BuildJobEmbeddingText(job),
	})
}

// Process generates an embedding when content changed; skips unchanged hashes.
func (s *EmbeddingService) Process(ctx context.Context, job EmbeddingJob) error {
	if s == nil || s.provider == nil {
		return fmt.Errorf("embedding service not configured")
	}
	text := strings.TrimSpace(job.Text)
	if text == "" {
		return nil
	}

	model := s.provider.Model()
	version := s.provider.Version()
	hash := embpkg.ContentHash(text, model, version)

	existing, err := s.repo.GetByEntity(ctx, job.EntityType, job.EntityID, model, version)
	if err == nil && existing.ContentHash == hash && existing.Status == domain.ProcessingStatusCompleted {
		return s.markParentStatus(ctx, job, domain.ProcessingStatusCompleted)
	}
	if err != nil && !errors.Is(err, repository.ErrEmbeddingNotFound) {
		return err
	}

	_ = s.markParentStatus(ctx, job, domain.ProcessingStatusProcessing)

	vector, err := s.embedWithRetry(ctx, text)
	if err != nil {
		_ = s.markParentStatus(ctx, job, domain.ProcessingStatusFailed)
		return fmt.Errorf("embed text: %w", err)
	}
	if len(vector) != s.provider.Dimensions() {
		_ = s.markParentStatus(ctx, job, domain.ProcessingStatusFailed)
		return fmt.Errorf("embedding dimension mismatch: got %d want %d", len(vector), s.provider.Dimensions())
	}

	now := time.Now().UTC()
	record := &domain.Embedding{
		CompanyID:        job.CompanyID,
		EntityType:       job.EntityType,
		EntityID:         job.EntityID,
		ContentHash:      hash,
		Embedding:        vector,
		EmbeddingModel:   model,
		EmbeddingVersion: version,
		EmbeddedAt:       now,
		Status:           domain.ProcessingStatusCompleted,
	}
	if err := s.repo.Upsert(ctx, record); err != nil {
		_ = s.markParentStatus(ctx, job, domain.ProcessingStatusFailed)
		return fmt.Errorf("store embedding: %w", err)
	}
	return s.markParentStatus(ctx, job, domain.ProcessingStatusCompleted)
}

func (s *EmbeddingService) embedWithRetry(ctx context.Context, text string) ([]float32, error) {
	const maxAttempts = 3
	backoff := 500 * time.Millisecond

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		vector, err := s.provider.Embed(ctx, text)
		if err == nil {
			return vector, nil
		}
		lastErr = err
		if !isRetryableEmbeddingError(err) || attempt == maxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	return nil, lastErr
}

func isRetryableEmbeddingError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "gemini embed api error (429)") ||
		strings.Contains(message, "gemini embed api error (500)") ||
		strings.Contains(message, "gemini embed api error (503)") ||
		strings.Contains(message, "context deadline exceeded") ||
		strings.Contains(message, "temporary")
}

func (s *EmbeddingService) markParentStatus(ctx context.Context, job EmbeddingJob, status string) error {
	switch job.EntityType {
	case domain.EmbeddingEntityCandidate:
		return s.repo.UpdateCandidateEmbeddingStatus(ctx, job.CompanyID, job.EntityID, status)
	case domain.EmbeddingEntityJob:
		return s.repo.UpdateJobEmbeddingStatus(ctx, job.CompanyID, job.EntityID, status)
	default:
		return nil
	}
}

// BuildCandidateEmbeddingText builds the canonical text for candidate vectors.
func BuildCandidateEmbeddingText(c *domain.Candidate) string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, 8)
	if c.CurrentDesignation != nil {
		parts = append(parts, *c.CurrentDesignation)
	}
	if c.CurrentCompany != nil {
		parts = append(parts, *c.CurrentCompany)
	}
	if c.Location != nil {
		parts = append(parts, *c.Location)
	}
	if c.ExperienceYears != nil {
		parts = append(parts, fmt.Sprintf("%d years experience", *c.ExperienceYears))
	}
	if len(c.Skills) > 0 {
		parts = append(parts, "Skills: "+strings.Join(c.Skills, ", "))
	}
	if c.ResumeSummary != nil && strings.TrimSpace(*c.ResumeSummary) != "" {
		parts = append(parts, *c.ResumeSummary)
	}
	if c.ResumeText != nil && strings.TrimSpace(*c.ResumeText) != "" {
		parts = append(parts, *c.ResumeText)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// BuildJobEmbeddingText builds the canonical text for job vectors.
func BuildJobEmbeddingText(j *domain.Job) string {
	if j == nil {
		return ""
	}
	parts := make([]string, 0, 8)
	parts = append(parts, j.Title)
	if j.Department != nil {
		parts = append(parts, *j.Department)
	}
	if j.Location != nil {
		parts = append(parts, *j.Location)
	}
	if j.ExperienceRequired != nil {
		parts = append(parts, "Experience: "+*j.ExperienceRequired)
	}
	if len(j.RequiredSkills) > 0 {
		parts = append(parts, "Required skills: "+strings.Join(j.RequiredSkills, ", "))
	}
	if j.Description != nil {
		parts = append(parts, *j.Description)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func (s *EmbeddingService) Model() string {
	if s == nil || s.provider == nil {
		return ""
	}
	return s.provider.Model()
}

func (s *EmbeddingService) Version() string {
	if s == nil || s.provider == nil {
		return ""
	}
	return s.provider.Version()
}

// ReindexJobsAndResumes regenerates embeddings for all jobs and parsed resumes
// using the currently configured provider/model/version.
func (s *EmbeddingService) ReindexJobsAndResumes(
	ctx context.Context,
	jobs []domain.Job,
	resumes []domain.Resume,
) (jobOK, resumeOK, failed int) {
	if s == nil {
		return 0, 0, 0
	}
	log.Printf("🧠 Reindex starting: %d jobs, %d resumes (model=%s version=%s)",
		len(jobs), len(resumes), s.Model(), s.Version())

	for i := range jobs {
		job := jobs[i]
		err := s.Process(ctx, EmbeddingJob{
			CompanyID:  job.CompanyID,
			EntityType: domain.EmbeddingEntityJob,
			EntityID:   job.ID,
			Text:       BuildJobEmbeddingText(&job),
		})
		if err != nil {
			failed++
			log.Printf("🧠 Reindex job %s failed: %v", job.ID, err)
			continue
		}
		jobOK++
		// Gentle pacing for remote embedding APIs.
		select {
		case <-ctx.Done():
			return jobOK, resumeOK, failed
		case <-time.After(50 * time.Millisecond):
		}
	}
	for i := range resumes {
		resume := resumes[i]
		text := ""
		if resume.ParsedText != nil {
			text = *resume.ParsedText
		}
		err := s.Process(ctx, EmbeddingJob{
			CompanyID:  resume.CompanyID,
			EntityType: domain.EmbeddingEntityResume,
			EntityID:   resume.ID,
			Text:       text,
		})
		if err != nil {
			failed++
			log.Printf("🧠 Reindex resume %s failed: %v", resume.ID, err)
			continue
		}
		resumeOK++
		select {
		case <-ctx.Done():
			return jobOK, resumeOK, failed
		case <-time.After(50 * time.Millisecond):
		}
	}

	log.Printf("🧠 Reindex finished: jobs_ok=%d resumes_ok=%d failed=%d", jobOK, resumeOK, failed)
	return jobOK, resumeOK, failed
}

// SemanticMatchesForJob finds the most similar candidates by resume↔job cosine similarity.
func (s *EmbeddingService) SemanticMatchesForJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	topK int,
) (*domain.SemanticMatchResult, error) {
	if s == nil || s.repo == nil || s.provider == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}
	if topK < 1 {
		topK = 20
	}
	if topK > 100 {
		topK = 100
	}

	result := &domain.SemanticMatchResult{
		JobID:   jobID,
		TopK:    topK,
		Matches: []domain.SemanticMatch{},
	}

	model := s.provider.Model()
	version := s.provider.Version()

	hasJob, err := s.repo.HasCompletedEntity(ctx, companyID, domain.EmbeddingEntityJob, jobID, model, version)
	if err != nil {
		return nil, err
	}
	if !hasJob {
		result.Status = domain.SemanticStatusJobEmbeddingMissing
		result.Message = "Job embedding is not ready yet. Save or update the job to generate an embedding, then retry."
		return result, nil
	}

	resumeCount, err := s.repo.CountCompletedByType(ctx, companyID, domain.EmbeddingEntityResume, model, version)
	if err != nil {
		return nil, err
	}

	candidateCount, err := s.repo.CountCompletedByType(ctx, companyID, domain.EmbeddingEntityCandidate, model, version)
	if err != nil {
		return nil, err
	}

	if resumeCount == 0 && candidateCount == 0 {
		result.Status = domain.SemanticStatusNoResumeEmbeddings
		result.Message = "No resume or candidate embeddings are available."
		return result, nil
	}

	var matches []domain.SemanticMatch

	if resumeCount > 0 {
		matches, err = s.repo.FindSimilarResumesToJob(ctx, companyID, jobID, model, version, topK)
	} else {
		matches, err = s.repo.FindSimilarCandidatesToJob(ctx, companyID, jobID, model, version, topK)
	}

	if err != nil {
		return nil, err
	}

	if matches == nil {
		matches = []domain.SemanticMatch{}
	}

	result.Matches = matches

	if len(matches) == 0 {
		result.Status = domain.SemanticStatusNoMatches
		result.Message = "No semantic matches found."
		return result, nil
	}

	result.Status = domain.SemanticStatusOK
	result.Message = "Semantic matches ordered by cosine distance (lower distance = higher similarity %)."
	return result, nil
}
