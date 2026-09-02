package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/similarity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

var ErrEmbeddingNotFound = errors.New("embedding not found")

type EmbeddingRepository struct {
	pool *pgxpool.Pool
}

func NewEmbeddingRepository(pool *pgxpool.Pool) *EmbeddingRepository {
	return &EmbeddingRepository{pool: pool}
}

func (r *EmbeddingRepository) GetByEntity(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	model, version string,
) (*domain.Embedding, error) {
	query := `
		SELECT id, company_id, entity_type, entity_id, content_hash,
			embedding, embedding_model, embedding_version, embedded_at, status,
			created_at, updated_at
		FROM embeddings
		WHERE entity_type = $1 AND entity_id = $2
			AND embedding_model = $3 AND embedding_version = $4
	`
	row := r.pool.QueryRow(ctx, query, entityType, entityID, model, version)
	emb, err := scanEmbedding(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEmbeddingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get embedding: %w", err)
	}
	return emb, nil
}

func (r *EmbeddingRepository) Upsert(ctx context.Context, emb *domain.Embedding) error {
	if emb.Embedding == nil {
		emb.Embedding = []float32{}
	}
	vec := pgvector.NewVector(emb.Embedding)

	query := `
		INSERT INTO embeddings (
			company_id, entity_type, entity_id, content_hash,
			embedding, embedding_model, embedding_version, embedded_at, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (entity_type, entity_id, embedding_model, embedding_version)
		DO UPDATE SET
			company_id = EXCLUDED.company_id,
			content_hash = EXCLUDED.content_hash,
			embedding = EXCLUDED.embedding,
			embedded_at = EXCLUDED.embedded_at,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		emb.CompanyID,
		emb.EntityType,
		emb.EntityID,
		emb.ContentHash,
		vec,
		emb.EmbeddingModel,
		emb.EmbeddingVersion,
		emb.EmbeddedAt,
		emb.Status,
	).Scan(&emb.ID, &emb.CreatedAt, &emb.UpdatedAt)
}

func (r *EmbeddingRepository) UpdateCandidateEmbeddingStatus(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	status string,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE candidates SET embedding_status = $1 WHERE id = $2 AND company_id = $3`,
		status, candidateID, companyID,
	)
	return err
}

func (r *EmbeddingRepository) UpdateJobEmbeddingStatus(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	status string,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET embedding_status = $1 WHERE id = $2 AND company_id = $3`,
		status, jobID, companyID,
	)
	return err
}

// HasCompletedEntity reports whether a completed embedding exists for the entity.
func (r *EmbeddingRepository) HasCompletedEntity(
	ctx context.Context,
	companyID uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	model, version string,
) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM embeddings
			WHERE company_id = $1
			  AND entity_type = $2
			  AND entity_id = $3
			  AND embedding_model = $4
			  AND embedding_version = $5
			  AND status = 'completed'
		)
	`, companyID, entityType, entityID, model, version).Scan(&exists)
	return exists, err
}

// CountCompletedByType counts completed embeddings for a company + entity type.
func (r *EmbeddingRepository) CountCompletedByType(
	ctx context.Context,
	companyID uuid.UUID,
	entityType, model, version string,
) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM embeddings
		WHERE company_id = $1
		  AND entity_type = $2
		  AND embedding_model = $3
		  AND embedding_version = $4
		  AND status = 'completed'
	`, companyID, entityType, model, version).Scan(&n)
	return n, err
}

// FindSimilarResumesToJob returns top-K candidates by cosine similarity of resume
// embeddings to the job embedding. Uses pgvector <=> (cosine distance) so the
// HNSW index on embeddings.embedding (vector_cosine_ops) can be used.
//
// Ranking uses raw cosine distance ascending (unchanged).
// similarity_score is the recruiter-facing percent:
//
//	percent = clamp(1 - (resume <=> job), 0, 1) × 100
func (r *EmbeddingRepository) FindSimilarResumesToJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	model, version string,
	topK int,
	candidateIDs []uuid.UUID,
) ([]domain.SemanticMatch, error) {
	if topK < 1 {
		topK = 20
	}
	// Job-scoped only: never search without an explicit applicant allow-list.
	if len(candidateIDs) == 0 {
		return []domain.SemanticMatch{}, nil
	}

	// Oversample so we can keep the best resume per candidate, then trim to topK.
	fetchLimit := topK * 5
	if fetchLimit < topK {
		fetchLimit = topK
	}
	if fetchLimit > 500 {
		fetchLimit = 500
	}

	query := `
		WITH job_emb AS (
			SELECT embedding
			FROM embeddings
			WHERE company_id = $1
			  AND entity_type = 'job'
			  AND entity_id = $2
			  AND embedding_model = $3
			  AND embedding_version = $4
			  AND status = 'completed'
			LIMIT 1
		)
		SELECT
			c.id AS candidate_id,
			c.name AS candidate_name,
			r.id AS resume_id,
			(e.embedding <=> je.embedding)::float8 AS cosine_distance,
			c.overall_score
		FROM embeddings e
		CROSS JOIN job_emb je
		INNER JOIN resumes r
			ON r.id = e.entity_id
		   AND r.company_id = e.company_id
		INNER JOIN candidates c
			ON c.id = r.candidate_id
		   AND c.company_id = e.company_id
		WHERE e.company_id = $1
		  AND e.entity_type = 'resume'
		  AND e.embedding_model = $3
		  AND e.embedding_version = $4
		  AND e.status = 'completed'
		  AND r.candidate_id IS NOT NULL
		  AND c.job_id = $2
		  AND c.id = ANY($5::uuid[])
		ORDER BY e.embedding <=> je.embedding ASC
		LIMIT $6
	`

	rows, err := r.pool.Query(ctx, query, companyID, jobID, model, version, candidateIDs, fetchLimit)
	if err != nil {
		return nil, fmt.Errorf("semantic resume search: %w", err)
	}
	defer rows.Close()

	seen := make(map[uuid.UUID]struct{})
	matches := make([]domain.SemanticMatch, 0, topK)
	for rows.Next() {
		var m domain.SemanticMatch
		var cosineDistance float64
		if err := rows.Scan(
			&m.CandidateID,
			&m.CandidateName,
			&m.ResumeID,
			&cosineDistance,
			&m.OverallScore,
		); err != nil {
			return nil, fmt.Errorf("scan semantic match: %w", err)
		}
		m.SimilarityScore = similarity.PercentFromCosineDistance(cosineDistance)
		if _, ok := seen[m.CandidateID]; ok {
			continue
		}
		seen[m.CandidateID] = struct{}{}
		matches = append(matches, m)
		if len(matches) >= topK {
			break
		}
	}
	return matches, rows.Err()
}
// FindSimilarResumesByVector returns top-K company resumes nearest to a query vector.
// Reuses existing embeddings only — does not regenerate stored vectors.
func (r *EmbeddingRepository) FindSimilarResumesByVector(
	ctx context.Context,
	companyID uuid.UUID,
	query []float32,
	model, version string,
	topK int,
) ([]domain.SemanticMatch, error) {
	if topK < 1 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	if len(query) == 0 {
		return []domain.SemanticMatch{}, nil
	}
	vec := pgvector.NewVector(query)
	querySQL := `
		SELECT
			c.id AS candidate_id,
			c.name AS candidate_name,
			r.id AS resume_id,
			(e.embedding <=> $1)::float8 AS cosine_distance,
			c.overall_score
		FROM embeddings e
		INNER JOIN resumes r
			ON r.id = e.entity_id
		   AND r.company_id = e.company_id
		INNER JOIN candidates c
			ON c.id = r.candidate_id
		   AND c.company_id = e.company_id
		WHERE e.company_id = $2
		  AND e.entity_type = 'resume'
		  AND e.embedding_model = $3
		  AND e.embedding_version = $4
		  AND e.status = 'completed'
		  AND r.candidate_id IS NOT NULL
		ORDER BY e.embedding <=> $1 ASC
		LIMIT $5
	`
	rows, err := r.pool.Query(ctx, querySQL, vec, companyID, model, version, topK)
	if err != nil {
		return nil, fmt.Errorf("query resume vector search: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.SemanticMatch, 0, topK)
	seen := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var m domain.SemanticMatch
		var cosineDistance float64
		if err := rows.Scan(
			&m.CandidateID,
			&m.CandidateName,
			&m.ResumeID,
			&cosineDistance,
			&m.OverallScore,
		); err != nil {
			return nil, fmt.Errorf("scan query resume match: %w", err)
		}
		if _, ok := seen[m.CandidateID]; ok {
			continue
		}
		seen[m.CandidateID] = struct{}{}
		m.SimilarityScore = similarity.PercentFromCosineDistance(cosineDistance)
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *EmbeddingRepository) FindSimilarCandidatesToJob(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	model, version string,
	topK int,
	candidateIDs []uuid.UUID,
) ([]domain.SemanticMatch, error) {
	if topK < 1 {
		topK = 20
	}
	if len(candidateIDs) == 0 {
		return []domain.SemanticMatch{}, nil
	}

	query := `
		WITH job_emb AS (
			SELECT embedding
			FROM embeddings
			WHERE company_id = $1
			  AND entity_type = 'job'
			  AND entity_id = $2
			  AND embedding_model = $3
			  AND embedding_version = $4
			  AND status = 'completed'
			LIMIT 1
		)
		SELECT
			c.id,
			c.name,
			(e.embedding <=> je.embedding)::float8 AS cosine_distance,
			c.overall_score
		FROM embeddings e
		CROSS JOIN job_emb je
		INNER JOIN candidates c
			ON c.id = e.entity_id
		   AND c.company_id = e.company_id
		WHERE e.company_id = $1
		  AND e.entity_type = 'candidate'
		  AND e.embedding_model = $3
		  AND e.embedding_version = $4
		  AND e.status = 'completed'
		  AND c.job_id = $2
		  AND c.id = ANY($5::uuid[])
		ORDER BY e.embedding <=> je.embedding
		LIMIT $6
	`

	rows, err := r.pool.Query(ctx, query, companyID, jobID, model, version, candidateIDs, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := []domain.SemanticMatch{}

	for rows.Next() {
		var m domain.SemanticMatch
		var cosineDistance float64

		if err := rows.Scan(
			&m.CandidateID,
			&m.CandidateName,
			&cosineDistance,
			&m.OverallScore,
		); err != nil {
			return nil, err
		}

		// Candidate-embedding matches have no resume row; leave ResumeID as uuid.Nil.
		m.ResumeID = uuid.Nil
		m.SimilarityScore = similarity.PercentFromCosineDistance(cosineDistance)
		matches = append(matches, m)
	}
	return matches, rows.Err()
}
func scanEmbedding(row scannable) (*domain.Embedding, error) {
	var (
		emb domain.Embedding
		vec pgvector.Vector
	)
	err := row.Scan(
		&emb.ID,
		&emb.CompanyID,
		&emb.EntityType,
		&emb.EntityID,
		&emb.ContentHash,
		&vec,
		&emb.EmbeddingModel,
		&emb.EmbeddingVersion,
		&emb.EmbeddedAt,
		&emb.Status,
		&emb.CreatedAt,
		&emb.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	emb.Embedding = vec.Slice()
	return &emb, nil
}

func FormatVectorLiteral(values []float32) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
