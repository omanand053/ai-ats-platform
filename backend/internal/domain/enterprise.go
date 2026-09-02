package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin         = "admin"
	RoleRecruiter     = "recruiter"
	RoleHiringManager = "hiring_manager"
	RoleInterviewer   = "interviewer"
	RoleViewer        = "viewer"
)

// CompanyAISettings are per-tenant Overall AI Match weights (admin configurable).
type CompanyAISettings struct {
	CompanyID            uuid.UUID  `json:"company_id"`
	WeightSemantic       float64    `json:"weight_semantic"`
	WeightSkills         float64    `json:"weight_skills"`
	WeightExperience     float64    `json:"weight_experience"`
	WeightEducation      float64    `json:"weight_education"`
	WeightProjects       float64    `json:"weight_projects"`
	ConfidenceThreshold  float64    `json:"confidence_threshold"`
	EligibilityThreshold float64    `json:"eligibility_threshold"`
	UpdatedBy            *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type Notification struct {
	ID         uuid.UUID  `json:"id"`
	CompanyID  uuid.UUID  `json:"company_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	EntityType *string    `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID           uuid.UUID      `json:"id"`
	CompanyID    uuid.UUID      `json:"company_id"`
	ActorUserID  *uuid.UUID     `json:"actor_user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *uuid.UUID     `json:"resource_id,omitempty"`
	Meta         map[string]any `json:"meta,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Interview struct {
	ID                uuid.UUID  `json:"id"`
	CompanyID         uuid.UUID  `json:"company_id"`
	CandidateID       uuid.UUID  `json:"candidate_id"`
	JobID             *uuid.UUID `json:"job_id,omitempty"`
	Title             string     `json:"title"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
	DurationMinutes   int        `json:"duration_minutes"`
	Timezone          string     `json:"timezone"`
	Location          *string    `json:"location,omitempty"`
	MeetingURL        *string    `json:"meeting_url,omitempty"`
	Status            string     `json:"status"`
	InterviewerUserID *uuid.UUID `json:"interviewer_user_id,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CandidateName     string     `json:"candidate_name,omitempty"`
}

type CollaborationComment struct {
	ID           uuid.UUID   `json:"id"`
	CompanyID    uuid.UUID   `json:"company_id"`
	CandidateID  uuid.UUID   `json:"candidate_id"`
	AuthorUserID *uuid.UUID  `json:"author_user_id,omitempty"`
	Body         string      `json:"body"`
	Mentions     []uuid.UUID `json:"mentions"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// AnalyticsOverview is the executive dashboard payload.
type AnalyticsOverview struct {
	TotalJobs             int64            `json:"total_jobs"`
	OpenJobs              int64            `json:"open_jobs"`
	ClosedJobs            int64            `json:"closed_jobs"`
	Applicants            int64            `json:"applicants"`
	Applications          int64            `json:"applications"`
	AIShortlisted         int64            `json:"ai_shortlisted"`
	RecruiterShortlisted  int64            `json:"recruiter_shortlisted"`
	Interviews            int64            `json:"interviews"`
	Offers                int64            `json:"offers"`
	Selected              int64            `json:"selected"`
	Rejected              int64            `json:"rejected"`
	Hired                 int64            `json:"hired"`
	AvgAIMatch            *float64         `json:"avg_ai_match,omitempty"`
	AvgTimeToHireDays     *float64         `json:"avg_time_to_hire_days,omitempty"`
	OfferAcceptanceRate   *float64         `json:"offer_acceptance_rate,omitempty"`
	ByStatus              map[string]int64 `json:"by_status"`
	ApplicationsPerJob    []NamedCount     `json:"applications_per_job"`
	TopSkills             []NamedCount     `json:"top_skills"`
	MissingSkills         []NamedCount     `json:"missing_skills"`
	AIMatchDistribution   []BucketCount    `json:"ai_match_distribution"`
	HiringTrend           []TrendPoint     `json:"hiring_trend"`
	MonthlyHiring         []TrendPoint     `json:"monthly_hiring"`
	RecruiterProductivity []NamedCount     `json:"recruiter_productivity"`
	Funnel                []NamedCount     `json:"funnel"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

type TrendPoint struct {
	Period string `json:"period"`
	Count  int64  `json:"count"`
}
