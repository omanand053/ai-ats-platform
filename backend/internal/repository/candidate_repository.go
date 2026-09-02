package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCandidateNotFound = errors.New("candidate not found")
var ErrCandidateEmailExists = errors.New("candidate email already exists")

type CandidateRepository struct {
	pool *pgxpool.Pool
}

func NewCandidateRepository(pool *pgxpool.Pool) *CandidateRepository {
	return &CandidateRepository{pool: pool}
}

const candidateColumns = `
	id, company_id, job_id, name, email, phone,
	experience_years, current_company, current_designation, location,
	skills, status, resume_url, resume_text, resume_summary, source,
	parsing_status, embedding_status,
	overall_score, score_breakdown, matched_skills, missing_skills, last_scored_at,
	created_at, updated_at
`

type CandidateListFilter struct {
	Status string
	Search string
	JobID  *uuid.UUID
	Sort   string
}

func (r *CandidateRepository) Create(ctx context.Context, candidate *domain.Candidate) error {
	skills := candidate.Skills
	if skills == nil {
		skills = []string{}
	}
	matched := candidate.MatchedSkills
	if matched == nil {
		matched = []string{}
	}
	missing := candidate.MissingSkills
	if missing == nil {
		missing = []string{}
	}
	breakdown, err := marshalScoreBreakdown(candidate.ScoreBreakdown)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO candidates (
			company_id, job_id, name, email, phone,
			experience_years, current_company, current_designation, location,
			skills, status, resume_url, resume_text, resume_summary, source,
			parsing_status, embedding_status,
			overall_score, score_breakdown, matched_skills, missing_skills, last_scored_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17,
			$18, $19, $20, $21, $22
		)
		RETURNING id, created_at, updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		candidate.CompanyID,
		candidate.JobID,
		candidate.Name,
		candidate.Email,
		candidate.Phone,
		candidate.ExperienceYears,
		candidate.CurrentCompany,
		candidate.CurrentDesignation,
		candidate.Location,
		skills,
		candidate.Status,
		candidate.ResumeURL,
		candidate.ResumeText,
		candidate.ResumeSummary,
		candidate.Source,
		candidate.ParsingStatus,
		candidate.EmbeddingStatus,
		candidate.OverallScore,
		breakdown,
		matched,
		missing,
		candidate.LastScoredAt,
	).Scan(&candidate.ID, &candidate.CreatedAt, &candidate.UpdatedAt)

	if err != nil && isUniqueViolation(err) {
		return ErrCandidateEmailExists
	}
	return err
}

func (r *CandidateRepository) GetByID(ctx context.Context, companyID, candidateID uuid.UUID) (*domain.Candidate, error) {
	query := `SELECT ` + candidateColumns + ` FROM candidates WHERE id = $1 AND company_id = $2`

	candidate := &domain.Candidate{}
	err := r.scanCandidate(r.pool.QueryRow(ctx, query, candidateID, companyID), candidate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCandidateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get candidate by id: %w", err)
	}

	return candidate, nil
}

func (r *CandidateRepository) List(
	ctx context.Context,
	companyID uuid.UUID,
	page, limit int,
	filter CandidateListFilter,
) (*domain.CandidateListResult, error) {
	offset := (page - 1) * limit

	where := `WHERE company_id = $1`
	args := []any{companyID}

	if filter.Status != "" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(` AND status = $%d`, len(args))
	}

	if filter.JobID != nil {
		args = append(args, *filter.JobID)
		where += fmt.Sprintf(` AND job_id = $%d`, len(args))
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		idx := len(args)
		where += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, idx, idx)
	}

	orderBy := ` ORDER BY created_at DESC`
	switch strings.ToLower(strings.TrimSpace(filter.Sort)) {
	case "score", "score_desc", "highest_score":
		orderBy = ` ORDER BY overall_score DESC NULLS LAST, created_at DESC`
	case "score_asc":
		orderBy = ` ORDER BY overall_score ASC NULLS LAST, created_at DESC`
	case "created_at", "newest", "":
		orderBy = ` ORDER BY created_at DESC`
	}

	countQuery := `SELECT COUNT(*) FROM candidates ` + where
	listQuery := `SELECT ` + candidateColumns + ` FROM candidates ` + where + orderBy
	listQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	listArgs := append(append([]any{}, args...), limit, offset)

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count candidates: %w", err)
	}

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.Candidate, 0)
	for rows.Next() {
		candidate := domain.Candidate{}
		if err := r.scanCandidate(rows, &candidate); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.CandidateListResult{
		Candidates: candidates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *CandidateRepository) ListByJobID(ctx context.Context, companyID, jobID uuid.UUID) ([]domain.Candidate, error) {
	query := `SELECT ` + candidateColumns + ` FROM candidates WHERE company_id = $1 AND job_id = $2 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, companyID, jobID)
	if err != nil {
		return nil, fmt.Errorf("list candidates by job: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.Candidate, 0)
	for rows.Next() {
		candidate := domain.Candidate{}
		if err := r.scanCandidate(rows, &candidate); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// ListForSemanticFiltering returns candidates for pre-filter evaluation.
// candidateIDs must be non-empty; an empty list returns no rows (never the company pool).
func (r *CandidateRepository) ListForSemanticFiltering(ctx context.Context, companyID uuid.UUID, candidateIDs []uuid.UUID) ([]domain.Candidate, error) {
	if len(candidateIDs) == 0 {
		return []domain.Candidate{}, nil
	}

	query := `SELECT ` + candidateColumns + `
		FROM candidates c
		WHERE c.company_id = $1
		  AND c.id = ANY($2::uuid[])`
	rows, err := r.pool.Query(ctx, query, companyID, candidateIDs)
	if err != nil {
		return nil, fmt.Errorf("list candidates for semantic filtering: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.Candidate, 0)
	for rows.Next() {
		candidate := domain.Candidate{}
		if err := r.scanCandidate(rows, &candidate); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// ListUnscoredWithJob returns candidates linked to a job that have no overall_score yet.
func (r *CandidateRepository) ListUnscoredWithJob(ctx context.Context) ([]domain.Candidate, error) {
	query := `SELECT ` + candidateColumns + `
		FROM candidates
		WHERE job_id IS NOT NULL AND overall_score IS NULL
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list unscored candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.Candidate, 0)
	for rows.Next() {
		candidate := domain.Candidate{}
		if err := r.scanCandidate(rows, &candidate); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func (r *CandidateRepository) Update(ctx context.Context, candidate *domain.Candidate) error {
	skills := candidate.Skills
	if skills == nil {
		skills = []string{}
	}
	matched := candidate.MatchedSkills
	if matched == nil {
		matched = []string{}
	}
	missing := candidate.MissingSkills
	if missing == nil {
		missing = []string{}
	}
	breakdown, err := marshalScoreBreakdown(candidate.ScoreBreakdown)
	if err != nil {
		return err
	}

	query := `
		UPDATE candidates SET
			job_id = $1,
			name = $2,
			email = $3,
			phone = $4,
			experience_years = $5,
			current_company = $6,
			current_designation = $7,
			location = $8,
			skills = $9,
			status = $10,
			resume_url = $11,
			resume_text = $12,
			resume_summary = $13,
			source = $14,
			parsing_status = $15,
			embedding_status = $16,
			overall_score = $17,
			score_breakdown = $18,
			matched_skills = $19,
			missing_skills = $20,
			last_scored_at = $21
		WHERE id = $22 AND company_id = $23
		RETURNING updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		candidate.JobID,
		candidate.Name,
		candidate.Email,
		candidate.Phone,
		candidate.ExperienceYears,
		candidate.CurrentCompany,
		candidate.CurrentDesignation,
		candidate.Location,
		skills,
		candidate.Status,
		candidate.ResumeURL,
		candidate.ResumeText,
		candidate.ResumeSummary,
		candidate.Source,
		candidate.ParsingStatus,
		candidate.EmbeddingStatus,
		candidate.OverallScore,
		breakdown,
		matched,
		missing,
		candidate.LastScoredAt,
		candidate.ID,
		candidate.CompanyID,
	).Scan(&candidate.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCandidateNotFound
	}
	if err != nil && isUniqueViolation(err) {
		return ErrCandidateEmailExists
	}
	return err
}

func (r *CandidateRepository) Delete(ctx context.Context, companyID, candidateID uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM candidates WHERE id = $1 AND company_id = $2`,
		candidateID, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete candidate: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCandidateNotFound
	}
	return nil
}

func (r *CandidateRepository) scanCandidate(row scannable, candidate *domain.Candidate) error {
	var breakdownJSON []byte
	err := row.Scan(
		&candidate.ID,
		&candidate.CompanyID,
		&candidate.JobID,
		&candidate.Name,
		&candidate.Email,
		&candidate.Phone,
		&candidate.ExperienceYears,
		&candidate.CurrentCompany,
		&candidate.CurrentDesignation,
		&candidate.Location,
		&candidate.Skills,
		&candidate.Status,
		&candidate.ResumeURL,
		&candidate.ResumeText,
		&candidate.ResumeSummary,
		&candidate.Source,
		&candidate.ParsingStatus,
		&candidate.EmbeddingStatus,
		&candidate.OverallScore,
		&breakdownJSON,
		&candidate.MatchedSkills,
		&candidate.MissingSkills,
		&candidate.LastScoredAt,
		&candidate.CreatedAt,
		&candidate.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if len(breakdownJSON) > 0 && string(breakdownJSON) != "null" {
		var bd domain.FitScoreBreakdown
		if err := json.Unmarshal(breakdownJSON, &bd); err != nil {
			return fmt.Errorf("unmarshal score_breakdown: %w", err)
		}
		candidate.ScoreBreakdown = &bd
	}
	if candidate.MatchedSkills == nil {
		candidate.MatchedSkills = []string{}
	}
	if candidate.MissingSkills == nil {
		candidate.MissingSkills = []string{}
	}
	return nil
}

func marshalScoreBreakdown(bd *domain.FitScoreBreakdown) ([]byte, error) {
	if bd == nil {
		return nil, nil
	}
	return json.Marshal(bd)
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate"))
}
