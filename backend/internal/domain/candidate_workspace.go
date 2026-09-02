package domain

import (
	"time"

	"github.com/google/uuid"
)

// CandidateNote is a recruiter note attached to a candidate.
type CandidateNote struct {
	ID           uuid.UUID  `json:"id"`
	CompanyID    uuid.UUID  `json:"company_id"`
	CandidateID  uuid.UUID  `json:"candidate_id"`
	AuthorUserID *uuid.UUID `json:"author_user_id,omitempty"`
	Body         string     `json:"body"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Timeline event types for the recruiter drawer.
const (
	TimelineApplied           = "applied"
	TimelineResumeParsed      = "resume_parsed"
	TimelineAIEvaluated       = "ai_evaluated"
	TimelineAIShortlisted     = "ai_shortlisted"
	TimelineRecruiterReviewed = "recruiter_reviewed"
	TimelineInterview         = "interview"
	TimelineSelected          = "selected"
	TimelineRejected          = "rejected"
	TimelineNoteAdded         = "note_added"
	TimelineStatusChanged     = "status_changed"
)

// CandidateTimelineItem is one step in the recruiter timeline.
type CandidateTimelineItem struct {
	ID        uuid.UUID `json:"id"`
	EventType string    `json:"event_type"`
	Label     string    `json:"label"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // "recorded" | "inferred"
}
