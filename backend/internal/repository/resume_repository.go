package repository

import (
	"context"
	"errors"
	"fmt"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrResumeNotFound = errors.New("resume not found")

type ResumeRepository struct {
	pool *pgxpool.Pool
}

func NewResumeRepository(pool *pgxpool.Pool) *ResumeRepository {
	return &ResumeRepository{pool: pool}
}

func (r *ResumeRepository) Create(ctx context.Context, resume *domain.Resume) error {
	query := `
		INSERT INTO resumes (
			candidate_id, uploaded_by, company_id, file_name, file_url, storage_path,
			file_size, mime_type, parsed_text, is_primary, parsing_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		resume.CandidateID,
		resume.UploadedBy,
		resume.CompanyID,
		resume.FileName,
		resume.FileURL,
		resume.StoragePath,
		resume.FileSize,
		resume.MimeType,
		resume.ParsedText,
		resume.IsPrimary,
		resume.ParsingStatus,
	).Scan(&resume.ID, &resume.CreatedAt, &resume.UpdatedAt)
}

func (r *ResumeRepository) CreateWithID(ctx context.Context, resume *domain.Resume) error {
	query := `
		INSERT INTO resumes (
			id, candidate_id, uploaded_by, company_id, file_name, file_url, storage_path,
			file_size, mime_type, parsed_text, is_primary, parsing_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		resume.ID,
		resume.CandidateID,
		resume.UploadedBy,
		resume.CompanyID,
		resume.FileName,
		resume.FileURL,
		resume.StoragePath,
		resume.FileSize,
		resume.MimeType,
		resume.ParsedText,
		resume.IsPrimary,
		resume.ParsingStatus,
	).Scan(&resume.CreatedAt, &resume.UpdatedAt)
}

func (r *ResumeRepository) UpdateParsed(ctx context.Context, id uuid.UUID, parsedText *string, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE resumes
		SET parsed_text = $2, parsing_status = $3, updated_at = NOW()
		WHERE id = $1
	`, id, parsedText, status)
	if err != nil {
		return fmt.Errorf("update resume parsed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResumeNotFound
	}
	return nil
}

func (r *ResumeRepository) AttachCandidate(ctx context.Context, resumeID, candidateID, companyID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE resumes
		SET candidate_id = $2, is_primary = TRUE, updated_at = NOW()
		WHERE id = $1 AND company_id = $3
		  AND EXISTS (
			SELECT 1 FROM candidates c
			WHERE c.id = $2 AND c.company_id = $3
		  )
	`, resumeID, candidateID, companyID)
	if err != nil {
		return fmt.Errorf("attach resume candidate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResumeNotFound
	}
	return nil
}

func (r *ResumeRepository) GetByID(ctx context.Context, id, companyID uuid.UUID) (*domain.Resume, error) {
	query := `
		SELECT id, candidate_id, uploaded_by, company_id, file_name, file_url, storage_path,
			file_size, mime_type, parsed_text, is_primary, parsing_status, created_at, updated_at
		FROM resumes
		WHERE id = $1 AND company_id = $2
	`

	resume := &domain.Resume{}
	err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&resume.ID,
		&resume.CandidateID,
		&resume.UploadedBy,
		&resume.CompanyID,
		&resume.FileName,
		&resume.FileURL,
		&resume.StoragePath,
		&resume.FileSize,
		&resume.MimeType,
		&resume.ParsedText,
		&resume.IsPrimary,
		&resume.ParsingStatus,
		&resume.CreatedAt,
		&resume.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrResumeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get resume: %w", err)
	}
	return resume, nil
}

func (r *ResumeRepository) GetByIDs(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*domain.Resume, error) {
	out := make(map[uuid.UUID]*domain.Resume, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, candidate_id, uploaded_by, company_id, file_name, file_url, storage_path,
			file_size, mime_type, parsed_text, is_primary, parsing_status, created_at, updated_at
		FROM resumes
		WHERE company_id = $1 AND id = ANY($2)
	`, companyID, ids)
	if err != nil {
		return nil, fmt.Errorf("get resumes by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		resume := &domain.Resume{}
		if err := rows.Scan(
			&resume.ID,
			&resume.CandidateID,
			&resume.UploadedBy,
			&resume.CompanyID,
			&resume.FileName,
			&resume.FileURL,
			&resume.StoragePath,
			&resume.FileSize,
			&resume.MimeType,
			&resume.ParsedText,
			&resume.IsPrimary,
			&resume.ParsingStatus,
			&resume.CreatedAt,
			&resume.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan resume: %w", err)
		}
		out[resume.ID] = resume
	}
	return out, rows.Err()
}

// ListParsedForEmbedding returns resumes that have non-empty parsed text.
func (r *ResumeRepository) ListParsedForEmbedding(ctx context.Context) ([]domain.Resume, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, candidate_id, uploaded_by, company_id, file_name, file_url, storage_path,
		       file_size, mime_type, parsed_text, is_primary, parsing_status, created_at, updated_at
		FROM resumes
		WHERE parsed_text IS NOT NULL
		  AND TRIM(parsed_text) <> ''
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list resumes for embedding: %w", err)
	}
	defer rows.Close()

	var out []domain.Resume
	for rows.Next() {
		var resume domain.Resume
		if err := rows.Scan(
			&resume.ID,
			&resume.CandidateID,
			&resume.UploadedBy,
			&resume.CompanyID,
			&resume.FileName,
			&resume.FileURL,
			&resume.StoragePath,
			&resume.FileSize,
			&resume.MimeType,
			&resume.ParsedText,
			&resume.IsPrimary,
			&resume.ParsingStatus,
			&resume.CreatedAt,
			&resume.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan resume for embedding: %w", err)
		}
		out = append(out, resume)
	}
	return out, rows.Err()
}
