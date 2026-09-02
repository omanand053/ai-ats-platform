package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobHandler struct {
	jobService         *service.JobService
	aiAssistantService *service.AIAssistantService
}

func NewJobHandler(jobService *service.JobService, aiAssistantService *service.AIAssistantService) *JobHandler {
	return &JobHandler{jobService: jobService, aiAssistantService: aiAssistantService}
}

type jobRequest struct {
	Title              string   `json:"title" binding:"required,min=1,max=255"`
	Department         string   `json:"department" binding:"omitempty,max=100"`
	Location           string   `json:"location" binding:"omitempty,max=255"`
	EmploymentType     string   `json:"employment_type" binding:"omitempty,max=50"`
	ExperienceRequired string   `json:"experience_required" binding:"omitempty,max=255"`
	Description        string   `json:"description"`
	RequiredSkills     []string `json:"required_skills"`
	Status             string   `json:"status" binding:"omitempty,max=50"`
}

func (h *JobHandler) Create(c *gin.Context) {
	companyID, userID, ok := h.authContext(c)
	if !ok {
		return
	}

	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	job, err := h.jobService.Create(c.Request.Context(), companyID, userID, req.toInput())
	if err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusCreated, gin.H{"job": job})
}

func (h *JobHandler) List(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	result, err := h.jobService.List(c.Request.Context(), companyID, page, limit, status)
	if err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *JobHandler) GetByID(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	job, err := h.jobService.GetByID(c.Request.Context(), companyID, jobID)
	if err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"job": job})
}

func (h *JobHandler) SemanticMatches(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	topK, _ := strconv.Atoi(c.DefaultQuery("top_k", "0"))
	if topK == 0 {
		topK, _ = strconv.Atoi(c.DefaultQuery("limit", "0"))
	}

	result, err := h.jobService.SemanticMatches(c.Request.Context(), companyID, jobID, topK)
	if err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *JobHandler) AIAssistant(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	var req struct {
		Type         string   `json:"type"`
		Question     string   `json:"question"`
		TopK         int      `json:"top_k"`
		CandidateID  string   `json:"candidate_id"`
		CandidateIDs []string `json:"candidate_ids"`
		EmailKind    string   `json:"email_kind"`
		Difficulty   string   `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if h.aiAssistantService == nil {
		respondError(c, http.StatusServiceUnavailable, "ai_unavailable", "AI assistant is not configured")
		return
	}

	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = "qa"
	}
	if typ == "qa" && strings.TrimSpace(req.Question) == "" {
		respondError(c, http.StatusBadRequest, "validation_error", "question is required")
		return
	}

	copilotReq := service.CopilotRequest{
		Type:       typ,
		Question:   req.Question,
		TopK:       req.TopK,
		EmailKind:  req.EmailKind,
		Difficulty: req.Difficulty,
	}
	if req.CandidateID != "" {
		parsed, err := uuid.Parse(req.CandidateID)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_candidate_id", "Invalid candidate ID")
			return
		}
		copilotReq.CandidateID = &parsed
	}
	for _, idStr := range req.CandidateIDs {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_candidate_id", "Invalid candidate ID in candidate_ids")
			return
		}
		copilotReq.CandidateIDs = append(copilotReq.CandidateIDs, parsed)
	}

	result, err := h.aiAssistantService.Copilot(c.Request.Context(), companyID, jobID, copilotReq)
	if err != nil {
		msg := err.Error()
		if msg == "question is required" || strings.Contains(msg, "required") || strings.Contains(msg, "unsupported") {
			respondError(c, http.StatusBadRequest, "validation_error", msg)
			return
		}
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *JobHandler) Update(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	job, err := h.jobService.Update(c.Request.Context(), companyID, jobID, req.toInput())
	if err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"job": job})
}

func (h *JobHandler) Delete(c *gin.Context) {
	companyID, _, ok := h.authContext(c)
	if !ok {
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid job ID")
		return
	}

	if err := h.jobService.Delete(c.Request.Context(), companyID, jobID); err != nil {
		handleJobError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func (h *JobHandler) authContext(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return uuid.Nil, uuid.Nil, false
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return uuid.Nil, uuid.Nil, false
	}

	return companyID, userID, true
}

func (r jobRequest) toInput() service.JobInput {
	return service.JobInput{
		Title:              r.Title,
		Department:         r.Department,
		Location:           r.Location,
		EmploymentType:     r.EmploymentType,
		ExperienceRequired: r.ExperienceRequired,
		Description:        r.Description,
		RequiredSkills:     r.RequiredSkills,
		Status:             r.Status,
	}
}

func handleJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrJobNotFound):
		respondError(c, http.StatusNotFound, "job_not_found", "Job not found")
	case errors.Is(err, service.ErrInvalidJobStatus):
		respondError(c, http.StatusBadRequest, "invalid_status", "Status must be draft, open, or closed")
	case errors.Is(err, service.ErrInvalidEmployment):
		respondError(c, http.StatusBadRequest, "invalid_employment_type", "Invalid employment type")
	default:
		if msg := err.Error(); msg == "title is required" {
			respondError(c, http.StatusBadRequest, "validation_error", msg)
			return
		}
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
