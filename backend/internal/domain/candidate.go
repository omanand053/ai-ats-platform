package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	CandidateStatusApplied              = "applied"
	CandidateStatusScreening            = "screening"
	CandidateStatusShortlisted          = "shortlisted"
	CandidateStatusRecruiterShortlisted = "recruiter_shortlisted"
	CandidateStatusAIShortlisted        = "ai_shortlisted"
	CandidateStatusInterview            = "interview"
	CandidateStatusSelected             = "selected"
	CandidateStatusOffer                = "offer"
	CandidateStatusHired                = "hired"
	CandidateStatusRejected             = "rejected"

	ProcessingStatusPending    = "pending"
	ProcessingStatusProcessing = "processing"
	ProcessingStatusCompleted  = "completed"
	ProcessingStatusFailed     = "failed"
)

type Candidate struct {
	ID                 uuid.UUID          `json:"id"`
	CompanyID          uuid.UUID          `json:"company_id"`
	JobID              *uuid.UUID         `json:"job_id,omitempty"`
	Name               string             `json:"name"`
	Email              string             `json:"email"`
	Phone              *string            `json:"phone,omitempty"`
	ExperienceYears    *int               `json:"experience_years,omitempty"`
	CurrentCompany     *string            `json:"current_company,omitempty"`
	CurrentDesignation *string            `json:"current_designation,omitempty"`
	Location           *string            `json:"location,omitempty"`
	Skills             []string           `json:"skills"`
	Status             string             `json:"status"`
	ResumeURL          *string            `json:"resume_url,omitempty"`
	ResumeText         *string            `json:"resume_text,omitempty"`
	ResumeSummary      *string            `json:"resume_summary,omitempty"`
	Source             *string            `json:"source,omitempty"`
	ParsingStatus      string             `json:"parsing_status"`
	EmbeddingStatus    string             `json:"embedding_status"`
	OverallScore       *float64           `json:"overall_score,omitempty"`
	ScoreBreakdown     *FitScoreBreakdown `json:"score_breakdown,omitempty"`
	MatchedSkills      []string           `json:"matched_skills,omitempty"`
	MissingSkills      []string           `json:"missing_skills,omitempty"`
	LastScoredAt       *time.Time         `json:"last_scored_at,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// FitScoreBreakdown holds per-dimension scores (each 0–100).
type FitScoreBreakdown struct {
	Skills         float64 `json:"skills"`
	Experience     float64 `json:"experience"`
	Education      float64 `json:"education"`
	Seniority      float64 `json:"seniority"`
	Location       float64 `json:"location"`
	Certifications float64 `json:"certifications"`
}

type CandidateListResult struct {
	Candidates []Candidate `json:"candidates"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}
