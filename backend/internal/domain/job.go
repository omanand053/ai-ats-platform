package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	JobStatusDraft  = "draft"
	JobStatusOpen   = "open"
	JobStatusClosed = "closed"

	EmploymentFullTime   = "full_time"
	EmploymentPartTime   = "part_time"
	EmploymentContract   = "contract"
	EmploymentInternship = "internship"
	EmploymentTemporary  = "temporary"
)

type Job struct {
	ID                 uuid.UUID  `json:"id"`
	CompanyID          uuid.UUID  `json:"company_id"`
	Title              string     `json:"title"`
	Department         *string    `json:"department,omitempty"`
	Location           *string    `json:"location,omitempty"`
	EmploymentType     string     `json:"employment_type"`
	ExperienceRequired *string    `json:"experience_required,omitempty"`
	Description        *string    `json:"description,omitempty"`
	RequiredSkills     []string   `json:"required_skills"`
	Status             string     `json:"status"`
	EmbeddingStatus    string     `json:"embedding_status"`
	CreatedBy          *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type JobListResult struct {
	Jobs       []Job `json:"jobs"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
