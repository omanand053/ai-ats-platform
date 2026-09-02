package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

var ErrCandidateNoteNotFound = errors.New("candidate note not found")

// CandidateWorkspaceService powers recruiter notes + timeline without changing core scoring.
type CandidateWorkspaceService struct {
	repo       *repository.CandidateWorkspaceRepository
	candidates *repository.CandidateRepository
	hooks      EnterpriseHooks
}

func NewCandidateWorkspaceService(
	repo *repository.CandidateWorkspaceRepository,
	candidates *repository.CandidateRepository,
) *CandidateWorkspaceService {
	return &CandidateWorkspaceService{repo: repo, candidates: candidates}
}

func (s *CandidateWorkspaceService) SetEnterpriseHooks(hooks EnterpriseHooks) {
	if s != nil {
		s.hooks = hooks
	}
}

func (s *CandidateWorkspaceService) ListNotes(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) ([]domain.CandidateNote, error) {
	if err := s.repo.EnsureCandidateExists(ctx, companyID, candidateID); err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}
	return s.repo.ListNotes(ctx, companyID, candidateID)
}

func (s *CandidateWorkspaceService) CreateNote(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	authorUserID *uuid.UUID,
	body string,
) (*domain.CandidateNote, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("note body is required")
	}
	if len(body) > 4000 {
		return nil, fmt.Errorf("note body too long")
	}
	if err := s.repo.EnsureCandidateExists(ctx, companyID, candidateID); err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}

	note := &domain.CandidateNote{
		CompanyID:    companyID,
		CandidateID:  candidateID,
		AuthorUserID: authorUserID,
		Body:         body,
	}
	if err := s.repo.CreateNote(ctx, note); err != nil {
		return nil, err
	}
	_ = s.repo.AddEvent(ctx, companyID, candidateID, domain.TimelineNoteAdded, "Recruiter note added", map[string]any{
		"note_id": note.ID.String(),
	})
	_ = s.repo.AddEvent(ctx, companyID, candidateID, domain.TimelineRecruiterReviewed, "Recruiter reviewed", nil)
	if s.hooks != nil {
		et := "candidate"
		cid := candidateID
		s.hooks.NotifyCompany(ctx, companyID, "note_added", "Note added",
			truncateStr(body, 120), &et, &cid)
	}
	return note, nil
}

func (s *CandidateWorkspaceService) DeleteNote(
	ctx context.Context,
	companyID, candidateID, noteID uuid.UUID,
) error {
	if err := s.repo.EnsureCandidateExists(ctx, companyID, candidateID); err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return ErrCandidateNotFound
		}
		return err
	}
	err := s.repo.DeleteNote(ctx, companyID, candidateID, noteID)
	if errors.Is(err, repository.ErrCandidateNoteNotFound) {
		return ErrCandidateNoteNotFound
	}
	return err
}

func (s *CandidateWorkspaceService) RecordStatusChange(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
	fromStatus, toStatus string,
) {
	if s == nil || s.repo == nil || fromStatus == toStatus {
		return
	}
	label := statusTimelineLabel(toStatus)
	eventType := statusTimelineType(toStatus)
	_ = s.repo.AddEvent(ctx, companyID, candidateID, eventType, label, map[string]any{
		"from": fromStatus,
		"to":   toStatus,
	})
}

func (s *CandidateWorkspaceService) RecordApplied(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) {
	if s == nil || s.repo == nil {
		return
	}
	_ = s.repo.AddEvent(ctx, companyID, candidateID, domain.TimelineApplied, "Applied", nil)
}

func (s *CandidateWorkspaceService) Timeline(
	ctx context.Context,
	companyID, candidateID uuid.UUID,
) ([]domain.CandidateTimelineItem, error) {
	if s.candidates == nil {
		return nil, fmt.Errorf("candidate repository not configured")
	}
	candidate, err := s.candidates.GetByID(ctx, companyID, candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}

	recorded, err := s.repo.ListEvents(ctx, companyID, candidateID)
	if err != nil {
		return nil, err
	}

	inferred := inferTimeline(candidate)
	merged := mergeTimeline(inferred, recorded)
	return merged, nil
}

func statusTimelineType(status string) string {
	switch status {
	case domain.CandidateStatusAIShortlisted:
		return domain.TimelineAIShortlisted
	case domain.CandidateStatusRecruiterShortlisted, domain.CandidateStatusShortlisted:
		return domain.TimelineRecruiterReviewed
	case domain.CandidateStatusInterview:
		return domain.TimelineInterview
	case domain.CandidateStatusSelected, domain.CandidateStatusOffer, domain.CandidateStatusHired:
		return domain.TimelineSelected
	case domain.CandidateStatusRejected:
		return domain.TimelineRejected
	default:
		return domain.TimelineStatusChanged
	}
}

func statusTimelineLabel(status string) string {
	switch status {
	case domain.CandidateStatusAIShortlisted:
		return "AI Shortlisted"
	case domain.CandidateStatusRecruiterShortlisted:
		return "Recruiter Reviewed"
	case domain.CandidateStatusShortlisted:
		return "Shortlisted"
	case domain.CandidateStatusInterview:
		return "Interview"
	case domain.CandidateStatusSelected:
		return "Selected"
	case domain.CandidateStatusOffer:
		return "Offer"
	case domain.CandidateStatusHired:
		return "Hired"
	case domain.CandidateStatusRejected:
		return "Rejected"
	case domain.CandidateStatusScreening:
		return "Screening"
	default:
		return "Status updated: " + strings.ReplaceAll(status, "_", " ")
	}
}

func inferTimeline(c *domain.Candidate) []domain.CandidateTimelineItem {
	if c == nil {
		return nil
	}
	items := make([]domain.CandidateTimelineItem, 0, 4)
	items = append(items, domain.CandidateTimelineItem{
		ID:        uuid.NewSHA1(uuid.NameSpaceOID, []byte(c.ID.String()+":applied")),
		EventType: domain.TimelineApplied,
		Label:     "Applied",
		Timestamp: c.CreatedAt,
		Source:    "inferred",
	})
	if c.ParsingStatus == domain.ProcessingStatusCompleted {
		ts := c.CreatedAt.Add(time.Minute)
		if c.UpdatedAt.After(c.CreatedAt) {
			ts = c.CreatedAt.Add(2 * time.Minute)
		}
		items = append(items, domain.CandidateTimelineItem{
			ID:        uuid.NewSHA1(uuid.NameSpaceOID, []byte(c.ID.String()+":parsed")),
			EventType: domain.TimelineResumeParsed,
			Label:     "Resume Parsed",
			Timestamp: ts,
			Source:    "inferred",
		})
	}
	if c.LastScoredAt != nil || c.EmbeddingStatus == domain.ProcessingStatusCompleted {
		ts := c.UpdatedAt
		if c.LastScoredAt != nil {
			ts = *c.LastScoredAt
		}
		items = append(items, domain.CandidateTimelineItem{
			ID:        uuid.NewSHA1(uuid.NameSpaceOID, []byte(c.ID.String()+":ai")),
			EventType: domain.TimelineAIEvaluated,
			Label:     "AI Evaluated",
			Timestamp: ts,
			Source:    "inferred",
		})
	}
	return items
}

func mergeTimeline(inferred, recorded []domain.CandidateTimelineItem) []domain.CandidateTimelineItem {
	seen := map[string]struct{}{}
	out := make([]domain.CandidateTimelineItem, 0, len(inferred)+len(recorded))

	key := func(item domain.CandidateTimelineItem) string {
		return item.EventType + "|" + item.Timestamp.UTC().Format(time.RFC3339)
	}

	for _, item := range inferred {
		k := item.EventType + "|inferred"
		if _, ok := seen[k]; ok {
			continue
		}
		// Prefer recorded counterparts for pipeline stages.
		hasRecorded := false
		for _, r := range recorded {
			if r.EventType == item.EventType {
				hasRecorded = true
				break
			}
		}
		if hasRecorded && (item.EventType == domain.TimelineApplied ||
			item.EventType == domain.TimelineAIShortlisted ||
			item.EventType == domain.TimelineInterview ||
			item.EventType == domain.TimelineSelected) {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, item)
	}
	for _, item := range recorded {
		k := key(item)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, item)
	}

	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Timestamp.Before(out[i].Timestamp) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
