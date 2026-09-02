package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ResumeParsingPending    = "pending"
	ResumeParsingProcessing = "processing"
	ResumeParsingCompleted  = "completed"
	ResumeParsingFailed     = "failed"
)

type Resume struct {
	ID            uuid.UUID  `json:"id"`
	CandidateID   *uuid.UUID `json:"candidate_id,omitempty"`
	UploadedBy    *uuid.UUID `json:"uploaded_by,omitempty"`
	CompanyID     uuid.UUID  `json:"company_id"`
	FileName      string     `json:"file_name"`
	FileURL       string     `json:"file_url"`
	StoragePath   string     `json:"-"`
	FileSize      int64      `json:"file_size"`
	MimeType      string     `json:"mime_type"`
	ParsedText    *string    `json:"parsed_text,omitempty"`
	IsPrimary     bool       `json:"is_primary"`
	ParsingStatus string     `json:"parsing_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ParsedProject struct {
	Name         string   `json:"name"`
	Technologies []string `json:"technologies"`
	Description  string   `json:"description"`
}

type ParsedEducation struct {
	School string `json:"school"`
	Degree string `json:"degree"`
	Branch string `json:"branch"`
	Years  string `json:"years"`
}

// ParsedResume holds structured fields extracted from resume text.
type ParsedResume struct {
	FullName           string            `json:"full_name"`
	Email              string            `json:"email"`
	Phone              string            `json:"phone"`
	ExperienceYears    *int              `json:"experience_years,omitempty"`
	Skills             []string          `json:"skills"`
	Projects           []ParsedProject   `json:"projects"`
	Education          []ParsedEducation `json:"education"`
	Certifications     []string          `json:"certifications"`
	CurrentCompany     string            `json:"current_company"`
	CurrentDesignation string            `json:"current_designation"`
	Location           string            `json:"location"`
	Summary            string            `json:"summary"`
	RawText            string            `json:"raw_text,omitempty"`
}
