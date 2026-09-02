package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/utils"

	"github.com/google/uuid"
)

const (
	MaxResumeBytes = 5 * 1024 * 1024
	ResumeUploadDir = "uploads/resumes"
)

var (
	ErrInvalidResumeFile = errors.New("invalid resume file")
	ErrResumeTooLarge    = errors.New("resume file too large")
	ErrResumeNotFound    = repository.ErrResumeNotFound
)

type ResumeUploadResult struct {
	ResumeID      uuid.UUID           `json:"resume_id"`
	FileName      string              `json:"file_name"`
	FileURL       string              `json:"file_url"`
	ParsingStatus string              `json:"parsing_status"`
	Parsed        domain.ParsedResume `json:"parsed"`
}

type ResumeService struct {
	resumes    *repository.ResumeRepository
	candidates *CandidateService
	embeddings *EmbeddingService
	hooks      EnterpriseHooks
}

func NewResumeService(
	resumes *repository.ResumeRepository,
	candidates *CandidateService,
	embeddings *EmbeddingService,
) *ResumeService {
	return &ResumeService{resumes: resumes, candidates: candidates, embeddings: embeddings}
}

func (s *ResumeService) SetEnterpriseHooks(hooks EnterpriseHooks) {
	if s != nil {
		s.hooks = hooks
	}
}

func (s *ResumeService) UploadAndParse(
	ctx context.Context,
	companyID, userID uuid.UUID,
	originalName string,
	mimeType string,
	data []byte,
) (*ResumeUploadResult, error) {
	if err := validateResumeFile(originalName, mimeType, int64(len(data))); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(ResumeUploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	resumeID := uuid.New()
	ext := strings.ToLower(filepath.Ext(originalName))
	storageName := resumeID.String() + ext
	storagePath := filepath.Join(ResumeUploadDir, storageName)

	if err := os.WriteFile(storagePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("store resume: %w", err)
	}

	fileURL := fmt.Sprintf("/api/v1/resumes/%s/file", resumeID.String())
	safeName := filepath.Base(originalName)

	resume := &domain.Resume{
		ID:            resumeID,
		UploadedBy:    &userID,
		CompanyID:     companyID,
		FileName:      safeName,
		FileURL:       fileURL,
		StoragePath:   storagePath,
		FileSize:      int64(len(data)),
		MimeType:      mimeType,
		IsPrimary:     true,
		ParsingStatus: domain.ResumeParsingProcessing,
	}

	// Insert with predetermined ID
	if err := s.insertWithID(ctx, resume); err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}

	text, err := utils.ExtractTextFromResume(safeName, data)
	parsed := domain.ParsedResume{Skills: []string{}, Projects: []domain.ParsedProject{}, Education: []domain.ParsedEducation{}, Certifications: []string{}}
	status := domain.ResumeParsingCompleted
	if err != nil || strings.TrimSpace(text) == "" {
		status = domain.ResumeParsingFailed
		_ = s.resumes.UpdateParsed(ctx, resumeID, nil, status)
		return &ResumeUploadResult{
			ResumeID:      resumeID,
			FileName:      safeName,
			FileURL:       fileURL,
			ParsingStatus: status,
			Parsed:        parsed,
		}, nil
	}

	parsed = utils.ParseResumeText(text)
	parsedText := parsed.RawText
	if err := s.resumes.UpdateParsed(ctx, resumeID, &parsedText, status); err != nil {
		return nil, err
	}

	if s.embeddings != nil && strings.TrimSpace(parsedText) != "" {
		s.embeddings.EnqueueResume(companyID, resumeID, parsedText)
	}

	if s.hooks != nil {
		et := "resume"
		rid := resumeID
		s.hooks.NotifyCompany(ctx, companyID, "resume_uploaded", "Resume uploaded",
			fmt.Sprintf("%s uploaded", safeName), &et, &rid)
		actor := userID
		_ = s.hooks.Audit(ctx, companyID, &actor, "resume.upload", "resume", &rid, map[string]any{
			"file_name": safeName, "parsing_status": status,
		})
	}

	return &ResumeUploadResult{
		ResumeID:      resumeID,
		FileName:      safeName,
		FileURL:       fileURL,
		ParsingStatus: status,
		Parsed:        parsed,
	}, nil
}

func (s *ResumeService) insertWithID(ctx context.Context, resume *domain.Resume) error {
	// Use repository create then... Create generates ID. Override via direct insert in repo.
	// Prefer dedicated method:
	return s.resumes.CreateWithID(ctx, resume)
}

func (s *ResumeService) GetFile(ctx context.Context, resumeID, companyID uuid.UUID) (*domain.Resume, []byte, error) {
	resume, err := s.resumes.GetByID(ctx, resumeID, companyID)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(resume.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read resume file: %w", err)
	}
	return resume, data, nil
}

func (s *ResumeService) AttachCandidate(ctx context.Context, resumeID, candidateID, companyID uuid.UUID) error {
	if err := s.resumes.AttachCandidate(ctx, resumeID, candidateID, companyID); err != nil {
		return err
	}

	resume, err := s.resumes.GetByID(ctx, resumeID, companyID)
	if err != nil {
		return err
	}

	// Keep Candidate ↔ Resume consistent: mirror file URL / parsed text onto the candidate.
	if s.candidates != nil {
		candidate, cerr := s.candidates.GetByID(ctx, companyID, candidateID)
		if cerr == nil && candidate != nil {
			input := CandidateInput{
				JobID:              candidate.JobID,
				Name:               candidate.Name,
				Email:              candidate.Email,
				Phone:              derefString(candidate.Phone),
				ExperienceYears:    candidate.ExperienceYears,
				CurrentCompany:     derefString(candidate.CurrentCompany),
				CurrentDesignation: derefString(candidate.CurrentDesignation),
				Location:           derefString(candidate.Location),
				Skills:             candidate.Skills,
				Status:             candidate.Status,
				ResumeURL:          resume.FileURL,
				ResumeText:         firstNonEmpty(derefString(resume.ParsedText), derefString(candidate.ResumeText)),
				ResumeSummary:      derefString(candidate.ResumeSummary),
				Source:             derefString(candidate.Source),
				ParsingStatus:      resume.ParsingStatus,
				EmbeddingStatus:    candidate.EmbeddingStatus,
			}
			if _, uerr := s.candidates.Update(ctx, companyID, candidateID, input); uerr != nil {
				// Attach already succeeded; surface rescore/sync issues without undoing the link.
				_ = s.candidates.RescoreCandidate(ctx, companyID, candidateID)
			}
			return nil
		}
		_ = s.candidates.RescoreCandidate(ctx, companyID, candidateID)
	}
	return nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func validateResumeFile(name, mime string, size int64) error {
	if size <= 0 {
		return ErrInvalidResumeFile
	}
	if size > MaxResumeBytes {
		return ErrResumeTooLarge
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".pdf" && ext != ".docx" && ext != ".txt" {
		return ErrInvalidResumeFile
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	allowed := map[string]bool{
		"": true,
		"application/pdf": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/octet-stream": true,
		"text/plain":               true,
	}
	if !allowed[mime] {
		return ErrInvalidResumeFile
	}
	return nil
}
