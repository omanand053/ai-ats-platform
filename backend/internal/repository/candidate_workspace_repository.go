package repository

import (
	"context"
	"encoding/json"
	"errors"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCandidateNoteNotFound = errors.New("candidate note not found")

type CandidateWorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewCandidateWorkspaceRepository(pool *pgxpool.Pool) *CandidateWorkspaceRepository {
	return &CandidateWorkspaceRepository{pool: pool}
}

func (r *CandidateWorkspaceRepository) ListNotes(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) ([]domain.CandidateNote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, candidate_id, author_user_id, body, created_at, updated_at
		FROM candidate_notes
		WHERE company_id = $1 AND candidate_id = $2
		ORDER BY created_at DESC
	`, companyID, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]domain.CandidateNote, 0)
	for rows.Next() {
		var n domain.CandidateNote
		if err := rows.Scan(
			&n.ID, &n.CompanyID, &n.CandidateID, &n.AuthorUserID,
			&n.Body, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *CandidateWorkspaceRepository) CreateNote(
	ctx context.Context,
	note *domain.CandidateNote,
) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO candidate_notes (company_id, candidate_id, author_user_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, note.CompanyID, note.CandidateID, note.AuthorUserID, note.Body).Scan(
		&note.ID, &note.CreatedAt, &note.UpdatedAt,
	)
}

func (r *CandidateWorkspaceRepository) DeleteNote(
	ctx context.Context,
	companyID, candidateID, noteID uuid.UUID,
) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM candidate_notes
		WHERE company_id = $1 AND candidate_id = $2 AND id = $3
	`, companyID, candidateID, noteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCandidateNoteNotFound
	}
	return nil
}

func (r *CandidateWorkspaceRepository) AddEvent(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	eventType, label string,
	meta map[string]any,
) error {
	var metaJSON []byte
	var err error
	if meta != nil {
		metaJSON, err = json.Marshal(meta)
		if err != nil {
			return err
		}
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO candidate_events (company_id, candidate_id, event_type, label, meta)
		VALUES ($1, $2, $3, $4, $5)
	`, companyID, candidateID, eventType, label, metaJSON)
	return err
}

func (r *CandidateWorkspaceRepository) ListEvents(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) ([]domain.CandidateTimelineItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, event_type, label, created_at
		FROM candidate_events
		WHERE company_id = $1 AND candidate_id = $2
		ORDER BY created_at ASC
	`, companyID, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.CandidateTimelineItem, 0)
	for rows.Next() {
		var item domain.CandidateTimelineItem
		if err := rows.Scan(&item.ID, &item.EventType, &item.Label, &item.Timestamp); err != nil {
			return nil, err
		}
		item.Source = "recorded"
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CandidateWorkspaceRepository) EnsureCandidateExists(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) error {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM candidates WHERE company_id = $1 AND id = $2
	`, companyID, candidateID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCandidateNotFound
	}
	return err
}
