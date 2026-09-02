package repository

import (
	"context"
	"errors"
	"fmt"
	"math"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJobNotFound = errors.New("job not found")

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

const jobColumns = `
	id, company_id, title, department, location, employment_type,
	experience_required, description, required_skills, status, embedding_status,
	created_by, created_at, updated_at
`

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	skills := job.RequiredSkills
	if skills == nil {
		skills = []string{}
	}

	query := `
		INSERT INTO jobs (
			company_id, title, department, location, employment_type,
			experience_required, description, required_skills, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		job.CompanyID,
		job.Title,
		job.Department,
		job.Location,
		job.EmploymentType,
		job.ExperienceRequired,
		job.Description,
		skills,
		job.Status,
		job.CreatedBy,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *JobRepository) GetByID(ctx context.Context, companyID, jobID uuid.UUID) (*domain.Job, error) {
	query := `SELECT ` + jobColumns + ` FROM jobs WHERE id = $1 AND company_id = $2`

	job := &domain.Job{}
	err := r.scanJob(r.pool.QueryRow(ctx, query, jobID, companyID), job)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job by id: %w", err)
	}

	return job, nil
}

func (r *JobRepository) List(
	ctx context.Context,
	companyID uuid.UUID,
	page, limit int,
	status string,
) (*domain.JobListResult, error) {
	offset := (page - 1) * limit

	countQuery := `SELECT COUNT(*) FROM jobs WHERE company_id = $1`
	listQuery := `SELECT ` + jobColumns + ` FROM jobs WHERE company_id = $1`
	args := []any{companyID}

	if status != "" {
		countQuery += ` AND status = $2`
		listQuery += ` AND status = $2`
		args = append(args, status)
	}

	listQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	listArgs := append(append([]any{}, args...), limit, offset)

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count jobs: %w", err)
	}

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]domain.Job, 0)
	for rows.Next() {
		job := domain.Job{}
		if err := r.scanJob(rows, &job); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.JobListResult{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	skills := job.RequiredSkills
	if skills == nil {
		skills = []string{}
	}

	query := `
		UPDATE jobs SET
			title = $1,
			department = $2,
			location = $3,
			employment_type = $4,
			experience_required = $5,
			description = $6,
			required_skills = $7,
			status = $8
		WHERE id = $9 AND company_id = $10
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		job.Title,
		job.Department,
		job.Location,
		job.EmploymentType,
		job.ExperienceRequired,
		job.Description,
		skills,
		job.Status,
		job.ID,
		job.CompanyID,
	).Scan(&job.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrJobNotFound
	}
	return err
}

func (r *JobRepository) Delete(ctx context.Context, companyID, jobID uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM jobs WHERE id = $1 AND company_id = $2`,
		jobID, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// ListAllForEmbedding returns every job needed to (re)build job embeddings.
func (r *JobRepository) ListAllForEmbedding(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, title, department, location, employment_type,
		       experience_required, description, required_skills, status,
		       embedding_status, created_by, created_at, updated_at
		FROM jobs
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list jobs for embedding: %w", err)
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		if err := r.scanJob(rows, &job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func (r *JobRepository) scanJob(row scannable, job *domain.Job) error {
	return row.Scan(
		&job.ID,
		&job.CompanyID,
		&job.Title,
		&job.Department,
		&job.Location,
		&job.EmploymentType,
		&job.ExperienceRequired,
		&job.Description,
		&job.RequiredSkills,
		&job.Status,
		&job.EmbeddingStatus,
		&job.CreatedBy,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
}
