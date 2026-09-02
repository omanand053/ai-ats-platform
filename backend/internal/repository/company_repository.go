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

var ErrCompanyNotFound = errors.New("company not found")

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

func (r *CompanyRepository) Create(ctx context.Context, tx pgx.Tx, company *domain.Company) error {
	query := `
		INSERT INTO companies (name, slug, website, description, logo_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_active, created_at, updated_at
	`

	row := tx.QueryRow(ctx, query,
		company.Name,
		company.Slug,
		company.Website,
		company.Description,
		company.LogoURL,
	)

	return row.Scan(&company.ID, &company.IsActive, &company.CreatedAt, &company.UpdatedAt)
}

func (r *CompanyRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

func (r *CompanyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	query := `
		SELECT id, name, slug, website, description, logo_url, is_active, created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	company := &domain.Company{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&company.ID,
		&company.Name,
		&company.Slug,
		&company.Website,
		&company.Description,
		&company.LogoURL,
		&company.IsActive,
		&company.CreatedAt,
		&company.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCompanyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get company by id: %w", err)
	}

	return company, nil
}
