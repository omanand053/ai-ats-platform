package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EnterpriseRepository struct {
	pool *pgxpool.Pool
}

func NewEnterpriseRepository(pool *pgxpool.Pool) *EnterpriseRepository {
	return &EnterpriseRepository{pool: pool}
}

func (r *EnterpriseRepository) GetAISettings(ctx context.Context, companyID uuid.UUID) (*domain.CompanyAISettings, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT company_id, weight_semantic, weight_skills, weight_experience, weight_education, weight_projects,
		       confidence_threshold, eligibility_threshold, updated_by, updated_at
		FROM company_ai_settings WHERE company_id = $1
	`, companyID)
	var s domain.CompanyAISettings
	err := row.Scan(
		&s.CompanyID, &s.WeightSemantic, &s.WeightSkills, &s.WeightExperience, &s.WeightEducation, &s.WeightProjects,
		&s.ConfidenceThreshold, &s.EligibilityThreshold, &s.UpdatedBy, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *EnterpriseRepository) UpsertAISettings(ctx context.Context, s *domain.CompanyAISettings) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO company_ai_settings (
			company_id, weight_semantic, weight_skills, weight_experience, weight_education, weight_projects,
			confidence_threshold, eligibility_threshold, updated_by, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (company_id) DO UPDATE SET
			weight_semantic = EXCLUDED.weight_semantic,
			weight_skills = EXCLUDED.weight_skills,
			weight_experience = EXCLUDED.weight_experience,
			weight_education = EXCLUDED.weight_education,
			weight_projects = EXCLUDED.weight_projects,
			confidence_threshold = EXCLUDED.confidence_threshold,
			eligibility_threshold = EXCLUDED.eligibility_threshold,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING updated_at
	`, s.CompanyID, s.WeightSemantic, s.WeightSkills, s.WeightExperience, s.WeightEducation, s.WeightProjects,
		s.ConfidenceThreshold, s.EligibilityThreshold, s.UpdatedBy).Scan(&s.UpdatedAt)
}

func (r *EnterpriseRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO notifications (company_id, user_id, type, title, body, entity_type, entity_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at
	`, n.CompanyID, n.UserID, n.Type, n.Title, n.Body, n.EntityType, n.EntityID).Scan(&n.ID, &n.CreatedAt)
}

func (r *EnterpriseRepository) ListNotifications(ctx context.Context, companyID, userID uuid.UUID, limit int) ([]domain.Notification, error) {
	if limit < 1 {
		limit = 30
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, user_id, type, title, body, entity_type, entity_id, read_at, created_at
		FROM notifications
		WHERE company_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, companyID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Notification, 0)
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.CompanyID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.EntityType, &n.EntityID, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *EnterpriseRepository) MarkNotificationRead(ctx context.Context, companyID, userID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE company_id = $1 AND user_id = $2 AND id = $3 AND read_at IS NULL
	`, companyID, userID, id)
	return err
}

func (r *EnterpriseRepository) MarkAllNotificationsRead(ctx context.Context, companyID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE company_id = $1 AND user_id = $2 AND read_at IS NULL
	`, companyID, userID)
	return err
}

func (r *EnterpriseRepository) CountUnread(ctx context.Context, companyID, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE company_id = $1 AND user_id = $2 AND read_at IS NULL
	`, companyID, userID).Scan(&n)
	return n, err
}

func (r *EnterpriseRepository) CreateAuditLog(ctx context.Context, a *domain.AuditLog) error {
	var meta []byte
	var err error
	if a.Meta != nil {
		meta, err = json.Marshal(a.Meta)
		if err != nil {
			return err
		}
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_logs (company_id, actor_user_id, action, resource_type, resource_id, meta)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at
	`, a.CompanyID, a.ActorUserID, a.Action, a.ResourceType, a.ResourceID, meta).Scan(&a.ID, &a.CreatedAt)
}

func (r *EnterpriseRepository) ListAuditLogs(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.AuditLog, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, actor_user_id, action, resource_type, resource_id, meta, created_at
		FROM audit_logs WHERE company_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.AuditLog, 0)
	for rows.Next() {
		var a domain.AuditLog
		var meta []byte
		if err := rows.Scan(&a.ID, &a.CompanyID, &a.ActorUserID, &a.Action, &a.ResourceType, &a.ResourceID, &meta, &a.CreatedAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &a.Meta)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *EnterpriseRepository) CreateInterview(ctx context.Context, iv *domain.Interview) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO interviews (
			company_id, candidate_id, job_id, title, scheduled_at, duration_minutes, timezone,
			location, meeting_url, status, interviewer_user_id, notes, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at
	`, iv.CompanyID, iv.CandidateID, iv.JobID, iv.Title, iv.ScheduledAt, iv.DurationMinutes, iv.Timezone,
		iv.Location, iv.MeetingURL, iv.Status, iv.InterviewerUserID, iv.Notes, iv.CreatedBy,
	).Scan(&iv.ID, &iv.CreatedAt, &iv.UpdatedAt)
}

func (r *EnterpriseRepository) ListInterviews(ctx context.Context, companyID uuid.UUID, from, to *time.Time) ([]domain.Interview, error) {
	q := `
		SELECT i.id, i.company_id, i.candidate_id, i.job_id, i.title, i.scheduled_at, i.duration_minutes,
		       i.timezone, i.location, i.meeting_url, i.status, i.interviewer_user_id, i.notes,
		       i.created_by, i.created_at, i.updated_at, COALESCE(c.name, '')
		FROM interviews i
		LEFT JOIN candidates c ON c.id = i.candidate_id
		WHERE i.company_id = $1`
	args := []any{companyID}
	if from != nil {
		args = append(args, *from)
		q += ` AND i.scheduled_at >= $2`
	}
	if to != nil {
		args = append(args, *to)
		if from != nil {
			q += ` AND i.scheduled_at <= $3`
		} else {
			q += ` AND i.scheduled_at <= $2`
		}
	}
	q += ` ORDER BY i.scheduled_at ASC LIMIT 500`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Interview, 0)
	for rows.Next() {
		var iv domain.Interview
		if err := rows.Scan(
			&iv.ID, &iv.CompanyID, &iv.CandidateID, &iv.JobID, &iv.Title, &iv.ScheduledAt, &iv.DurationMinutes,
			&iv.Timezone, &iv.Location, &iv.MeetingURL, &iv.Status, &iv.InterviewerUserID, &iv.Notes,
			&iv.CreatedBy, &iv.CreatedAt, &iv.UpdatedAt, &iv.CandidateName,
		); err != nil {
			return nil, err
		}
		out = append(out, iv)
	}
	return out, rows.Err()
}

func (r *EnterpriseRepository) CreateComment(ctx context.Context, c *domain.CollaborationComment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO collaboration_comments (company_id, candidate_id, author_user_id, body, mentions)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at
	`, c.CompanyID, c.CandidateID, c.AuthorUserID, c.Body, c.Mentions).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *EnterpriseRepository) ListComments(ctx context.Context, companyID, candidateID uuid.UUID) ([]domain.CollaborationComment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, candidate_id, author_user_id, body, COALESCE(mentions, '{}'), created_at, updated_at
		FROM collaboration_comments
		WHERE company_id = $1 AND candidate_id = $2
		ORDER BY created_at DESC
	`, companyID, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CollaborationComment, 0)
	for rows.Next() {
		var c domain.CollaborationComment
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.CandidateID, &c.AuthorUserID, &c.Body, &c.Mentions, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if c.Mentions == nil {
			c.Mentions = []uuid.UUID{}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *EnterpriseRepository) AssignCandidate(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	assignee *uuid.UUID,
) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE candidates SET assigned_to = $1, updated_at = NOW()
		WHERE id = $2 AND company_id = $3
	`, assignee, candidateID, companyID)
	return err
}

func (r *EnterpriseRepository) ListCompanyUserIDs(ctx context.Context, companyID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM users WHERE company_id = $1 AND is_active = TRUE`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
