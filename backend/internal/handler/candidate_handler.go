package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CandidateHandler struct {
	candidateService *service.CandidateService
	workspaceService *service.CandidateWorkspaceService
}

func NewCandidateHandler(
	candidateService *service.CandidateService,
	workspaceService *service.CandidateWorkspaceService,
) *CandidateHandler {
	return &CandidateHandler{
		candidateService: candidateService,
		workspaceService: workspaceService,
	}
}

type candidateRequest struct {
	JobID              *uuid.UUID `json:"job_id"`
	Name               string     `json:"name" binding:"required,min=1,max=255"`
	Email              string     `json:"email" binding:"required,email,max=320"`
	Phone              string     `json:"phone" binding:"omitempty,max=50"`
	ExperienceYears    *int       `json:"experience_years" binding:"omitempty,min=0,max=80"`
	CurrentCompany     string     `json:"current_company" binding:"omitempty,max=255"`
	CurrentDesignation string     `json:"current_designation" binding:"omitempty,max=255"`
	Location           string     `json:"location" binding:"omitempty,max=255"`
	Skills             []string   `json:"skills"`
	Status             string     `json:"status" binding:"omitempty,max=50"`
	ResumeURL          string     `json:"resume_url" binding:"omitempty,max=512"`
	ResumeText         string     `json:"resume_text"`
	ResumeSummary      string     `json:"resume_summary"`
	Source             string     `json:"source" binding:"omitempty,max=100"`
	ParsingStatus      string     `json:"parsing_status" binding:"omitempty,max=50"`
	EmbeddingStatus    string     `json:"embedding_status" binding:"omitempty,max=50"`
}

func (h *CandidateHandler) Create(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	var req candidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	candidate, err := h.candidateService.Create(c.Request.Context(), companyID, req.toInput())
	if err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusCreated, gin.H{"candidate": candidate})
}

func (h *CandidateHandler) List(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")

	var jobID *uuid.UUID
	if jobIDStr := c.Query("job_id"); jobIDStr != "" {
		parsed, err := uuid.Parse(jobIDStr)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_job_id", "Invalid job ID")
			return
		}
		jobID = &parsed
	}

	result, err := h.candidateService.List(c.Request.Context(), companyID, page, limit, status, search, jobID, sort)
	if err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *CandidateHandler) ListByJob(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "score")

	result, err := h.candidateService.List(c.Request.Context(), companyID, page, limit, status, search, &jobID, sort)
	if err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *CandidateHandler) GetByID(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}

	candidate, err := h.candidateService.GetByID(c.Request.Context(), companyID, candidateID)
	if err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"candidate": candidate})
}

func (h *CandidateHandler) Update(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}

	var req candidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	candidate, err := h.candidateService.Update(c.Request.Context(), companyID, candidateID, req.toInput())
	if err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"candidate": candidate})
}

func (h *CandidateHandler) Delete(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}

	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}

	if err := h.candidateService.Delete(c.Request.Context(), companyID, candidateID); err != nil {
		handleCandidateError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Candidate deleted successfully"})
}

func (h *CandidateHandler) companyContext(c *gin.Context) (uuid.UUID, bool) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return uuid.Nil, false
	}
	return companyID, true
}

func (r candidateRequest) toInput() service.CandidateInput {
	return service.CandidateInput{
		JobID:              r.JobID,
		Name:               r.Name,
		Email:              r.Email,
		Phone:              r.Phone,
		ExperienceYears:    r.ExperienceYears,
		CurrentCompany:     r.CurrentCompany,
		CurrentDesignation: r.CurrentDesignation,
		Location:           r.Location,
		Skills:             r.Skills,
		Status:             r.Status,
		ResumeURL:          r.ResumeURL,
		ResumeText:         r.ResumeText,
		ResumeSummary:      r.ResumeSummary,
		Source:             r.Source,
		ParsingStatus:      r.ParsingStatus,
		EmbeddingStatus:    r.EmbeddingStatus,
	}
}

func handleCandidateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCandidateNotFound):
		respondError(c, http.StatusNotFound, "candidate_not_found", "Candidate not found")
	case errors.Is(err, service.ErrCandidateEmailExists):
		respondError(c, http.StatusConflict, "email_exists", "Candidate email already exists for this company")
	case errors.Is(err, service.ErrInvalidCandidateStatus):
		respondError(c, http.StatusBadRequest, "invalid_status", "Invalid candidate status")
	case errors.Is(err, service.ErrInvalidStatusTransition):
		respondError(c, http.StatusBadRequest, "invalid_status_transition", "That status change is not allowed for the current pipeline stage")
	case errors.Is(err, service.ErrInvalidJobReference):
		respondError(c, http.StatusBadRequest, "invalid_job_id", "Job not found in your company")
	default:
		if msg := err.Error(); msg == "name is required" || msg == "email is required" {
			respondError(c, http.StatusBadRequest, "validation_error", msg)
			return
		}
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
