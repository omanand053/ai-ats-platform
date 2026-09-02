package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

type EnterpriseService struct {
	repo      *repository.EnterpriseRepository
	analytics *repository.AnalyticsRepository
	defaults  config.AIMatchWeights
	filterMin float64
}

func NewEnterpriseService(
	repo *repository.EnterpriseRepository,
	analytics *repository.AnalyticsRepository,
	defaults config.AIMatchWeights,
	filterMin float64,
) *EnterpriseService {
	return &EnterpriseService{repo: repo, analytics: analytics, defaults: defaults, filterMin: filterMin}
}

func (s *EnterpriseService) Overview(ctx context.Context, companyID uuid.UUID) (*domain.AnalyticsOverview, error) {
	return s.analytics.Overview(ctx, companyID)
}

func (s *EnterpriseService) GetAISettings(ctx context.Context, companyID uuid.UUID) (*domain.CompanyAISettings, error) {
	row, err := s.repo.GetAISettings(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if row != nil {
		return row, nil
	}
	return &domain.CompanyAISettings{
		CompanyID:            companyID,
		WeightSemantic:       s.defaults.Semantic,
		WeightSkills:         s.defaults.Skills,
		WeightExperience:     s.defaults.Experience,
		WeightEducation:      s.defaults.Education,
		WeightProjects:       s.defaults.Projects,
		ConfidenceThreshold:  55,
		EligibilityThreshold: s.filterMin,
		UpdatedAt:            time.Now().UTC(),
	}, nil
}

func (s *EnterpriseService) UpdateAISettings(
	ctx context.Context,
	companyID uuid.UUID,
	actor *uuid.UUID,
	in domain.CompanyAISettings,
) (*domain.CompanyAISettings, error) {
	in.CompanyID = companyID
	in.UpdatedBy = actor
	normalizeAISettings(&in)
	if err := s.repo.UpsertAISettings(ctx, &in); err != nil {
		return nil, err
	}
	_ = s.Audit(ctx, companyID, actor, "ai_settings.update", "company_ai_settings", &companyID, map[string]any{
		"weights": map[string]float64{
			"semantic": in.WeightSemantic, "skills": in.WeightSkills,
			"experience": in.WeightExperience, "education": in.WeightEducation, "projects": in.WeightProjects,
		},
	})
	return &in, nil
}

func (s *EnterpriseService) ResolveAIWeights(ctx context.Context, companyID uuid.UUID) config.AIMatchWeights {
	row, err := s.repo.GetAISettings(ctx, companyID)
	if err != nil || row == nil {
		return s.defaults
	}
	w := config.AIMatchWeights{
		Semantic: row.WeightSemantic, Skills: row.WeightSkills, Experience: row.WeightExperience,
		Education: row.WeightEducation, Projects: row.WeightProjects,
	}
	sum := w.Semantic + w.Skills + w.Experience + w.Education + w.Projects
	if sum <= 0 {
		return s.defaults
	}
	w.Semantic /= sum
	w.Skills /= sum
	w.Experience /= sum
	w.Education /= sum
	w.Projects /= sum
	return w
}

func (s *EnterpriseService) ResolveEligibilityThreshold(ctx context.Context, companyID uuid.UUID) float64 {
	row, err := s.repo.GetAISettings(ctx, companyID)
	if err != nil || row == nil {
		return s.filterMin
	}
	return row.EligibilityThreshold
}

func (s *EnterpriseService) ListNotifications(ctx context.Context, companyID, userID uuid.UUID) ([]domain.Notification, error) {
	return s.repo.ListNotifications(ctx, companyID, userID, 40)
}

func (s *EnterpriseService) UnreadCount(ctx context.Context, companyID, userID uuid.UUID) (int64, error) {
	return s.repo.CountUnread(ctx, companyID, userID)
}

func (s *EnterpriseService) MarkRead(ctx context.Context, companyID, userID, id uuid.UUID) error {
	return s.repo.MarkNotificationRead(ctx, companyID, userID, id)
}

func (s *EnterpriseService) MarkAllRead(ctx context.Context, companyID, userID uuid.UUID) error {
	return s.repo.MarkAllNotificationsRead(ctx, companyID, userID)
}

func (s *EnterpriseService) NotifyCompany(
	ctx context.Context,
	companyID uuid.UUID,
	typ, title, body string,
	entityType *string,
	entityID *uuid.UUID,
) {
	ids, err := s.repo.ListCompanyUserIDs(ctx, companyID)
	if err != nil {
		return
	}
	for _, uid := range ids {
		n := &domain.Notification{
			CompanyID: companyID, UserID: uid, Type: typ, Title: title, Body: body,
			EntityType: entityType, EntityID: entityID,
		}
		_ = s.repo.CreateNotification(ctx, n)
	}
}

func (s *EnterpriseService) Audit(
	ctx context.Context,
	companyID uuid.UUID,
	actor *uuid.UUID,
	action, resourceType string,
	resourceID *uuid.UUID,
	meta map[string]any,
) error {
	return s.repo.CreateAuditLog(ctx, &domain.AuditLog{
		CompanyID: companyID, ActorUserID: actor, Action: action,
		ResourceType: resourceType, ResourceID: resourceID, Meta: meta,
	})
}

func (s *EnterpriseService) ListAuditLogs(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, companyID, limit, offset)
}

func (s *EnterpriseService) CreateInterview(ctx context.Context, iv *domain.Interview, actor *uuid.UUID) (*domain.Interview, error) {
	if iv.DurationMinutes < 15 {
		iv.DurationMinutes = 45
	}
	if strings.TrimSpace(iv.Timezone) == "" {
		iv.Timezone = "UTC"
	}
	if strings.TrimSpace(iv.Title) == "" {
		iv.Title = "Interview"
	}
	if iv.Status == "" {
		iv.Status = "scheduled"
	}
	iv.CreatedBy = actor
	if err := s.repo.CreateInterview(ctx, iv); err != nil {
		return nil, err
	}
	et := "interview"
	s.NotifyCompany(ctx, iv.CompanyID, "interview_scheduled", "Interview scheduled",
		fmt.Sprintf("%s · %s", iv.Title, iv.ScheduledAt.Format(time.RFC822)), &et, &iv.ID)
	_ = s.Audit(ctx, iv.CompanyID, actor, "interview.create", "interview", &iv.ID, map[string]any{
		"candidate_id": iv.CandidateID.String(),
		"scheduled_at": iv.ScheduledAt,
	})
	return iv, nil
}

func (s *EnterpriseService) ListInterviews(ctx context.Context, companyID uuid.UUID, from, to *time.Time) ([]domain.Interview, error) {
	return s.repo.ListInterviews(ctx, companyID, from, to)
}

func (s *EnterpriseService) ListComments(ctx context.Context, companyID, candidateID uuid.UUID) ([]domain.CollaborationComment, error) {
	return s.repo.ListComments(ctx, companyID, candidateID)
}

func (s *EnterpriseService) CreateComment(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	author *uuid.UUID,
	body string,
	mentions []uuid.UUID,
) (*domain.CollaborationComment, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("comment body is required")
	}
	c := &domain.CollaborationComment{
		CompanyID: companyID, CandidateID: candidateID, AuthorUserID: author, Body: body, Mentions: mentions,
	}
	if c.Mentions == nil {
		c.Mentions = []uuid.UUID{}
	}
	if err := s.repo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	et := "candidate"
	s.NotifyCompany(ctx, companyID, "comment_added", "New collaboration comment", truncateStr(body, 120), &et, &candidateID)
	_ = s.Audit(ctx, companyID, author, "comment.create", "candidate", &candidateID, nil)
	return c, nil
}

func (s *EnterpriseService) AssignCandidate(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	actor *uuid.UUID,
	assignee *uuid.UUID,
) error {
	if err := s.repo.AssignCandidate(ctx, companyID, candidateID, assignee); err != nil {
		return err
	}
	et := "candidate"
	if assignee != nil {
		s.NotifyCompany(ctx, companyID, "assignment", "Candidate assigned",
			"A candidate was assigned to a teammate", &et, &candidateID)
	}
	meta := map[string]any{}
	if assignee != nil {
		meta["assigned_to"] = assignee.String()
	}
	_ = s.Audit(ctx, companyID, actor, "candidate.assign", "candidate", &candidateID, meta)
	return nil
}

func normalizeAISettings(s *domain.CompanyAISettings) {
	sum := s.WeightSemantic + s.WeightSkills + s.WeightExperience + s.WeightEducation + s.WeightProjects
	if sum <= 0 {
		s.WeightSemantic, s.WeightSkills, s.WeightExperience, s.WeightEducation, s.WeightProjects = 0.4, 0.25, 0.15, 0.1, 0.1
		sum = 1
	}
	s.WeightSemantic /= sum
	s.WeightSkills /= sum
	s.WeightExperience /= sum
	s.WeightEducation /= sum
	s.WeightProjects /= sum
	if s.ConfidenceThreshold < 0 {
		s.ConfidenceThreshold = 0
	}
	if s.ConfidenceThreshold > 100 {
		s.ConfidenceThreshold = 100
	}
	if s.EligibilityThreshold < 0 {
		s.EligibilityThreshold = 0
	}
	if s.EligibilityThreshold > 100 {
		s.EligibilityThreshold = 100
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
