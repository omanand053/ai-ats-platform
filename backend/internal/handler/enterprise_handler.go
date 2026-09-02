package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EnterpriseHandler struct {
	svc *service.EnterpriseService
}

func NewEnterpriseHandler(svc *service.EnterpriseService) *EnterpriseHandler {
	return &EnterpriseHandler{svc: svc}
}

func (h *EnterpriseHandler) Overview(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	data, err := h.svc.Overview(c.Request.Context(), companyID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load analytics")
		return
	}
	respondSuccess(c, http.StatusOK, data)
}

func (h *EnterpriseHandler) GetAISettings(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	data, err := h.svc.GetAISettings(c.Request.Context(), companyID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load AI settings")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"settings": data})
}

func (h *EnterpriseHandler) UpdateAISettings(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	var req domain.CompanyAISettings
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	var actor *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		actor = &uid
	}
	data, err := h.svc.UpdateAISettings(c.Request.Context(), companyID, actor, req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to save AI settings")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"settings": data})
}

func (h *EnterpriseHandler) ListNotifications(c *gin.Context) {
	companyID, userID, ok := h.auth(c)
	if !ok {
		return
	}
	list, err := h.svc.ListNotifications(c.Request.Context(), companyID, userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load notifications")
		return
	}
	unread, _ := h.svc.UnreadCount(c.Request.Context(), companyID, userID)
	respondSuccess(c, http.StatusOK, gin.H{"notifications": list, "unread": unread})
}

func (h *EnterpriseHandler) MarkNotificationRead(c *gin.Context) {
	companyID, userID, ok := h.auth(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid notification ID")
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), companyID, userID, id); err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to update notification")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "ok"})
}

func (h *EnterpriseHandler) MarkAllNotificationsRead(c *gin.Context) {
	companyID, userID, ok := h.auth(c)
	if !ok {
		return
	}
	if err := h.svc.MarkAllRead(c.Request.Context(), companyID, userID); err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to update notifications")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "ok"})
}

func (h *EnterpriseHandler) ListAuditLogs(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	logs, err := h.svc.ListAuditLogs(c.Request.Context(), companyID, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load audit logs")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"logs": logs})
}

func (h *EnterpriseHandler) ListInterviews(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	var from, to *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = &t
		}
	}
	list, err := h.svc.ListInterviews(c.Request.Context(), companyID, from, to)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load interviews")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"interviews": list})
}

func (h *EnterpriseHandler) CreateInterview(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	var req struct {
		CandidateID       string  `json:"candidate_id" binding:"required"`
		JobID             string  `json:"job_id"`
		Title             string  `json:"title"`
		ScheduledAt       string  `json:"scheduled_at" binding:"required"`
		DurationMinutes   int     `json:"duration_minutes"`
		Timezone          string  `json:"timezone"`
		Location          string  `json:"location"`
		MeetingURL        string  `json:"meeting_url"`
		InterviewerUserID string  `json:"interviewer_user_id"`
		Notes             string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	candidateID, err := uuid.Parse(req.CandidateID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_candidate_id", "Invalid candidate ID")
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_scheduled_at", "scheduled_at must be RFC3339")
		return
	}
	iv := &domain.Interview{
		CompanyID: companyID, CandidateID: candidateID, Title: req.Title,
		ScheduledAt: scheduledAt, DurationMinutes: req.DurationMinutes, Timezone: req.Timezone,
		Status: "scheduled",
	}
	if req.JobID != "" {
		if id, err := uuid.Parse(req.JobID); err == nil {
			iv.JobID = &id
		}
	}
	if strings.TrimSpace(req.Location) != "" {
		v := req.Location
		iv.Location = &v
	}
	if strings.TrimSpace(req.MeetingURL) != "" {
		v := req.MeetingURL
		iv.MeetingURL = &v
	}
	if strings.TrimSpace(req.Notes) != "" {
		v := req.Notes
		iv.Notes = &v
	}
	if req.InterviewerUserID != "" {
		if id, err := uuid.Parse(req.InterviewerUserID); err == nil {
			iv.InterviewerUserID = &id
		}
	}
	var actor *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		actor = &uid
	}
	created, err := h.svc.CreateInterview(c.Request.Context(), iv, actor)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to create interview")
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{"interview": created})
}

func (h *EnterpriseHandler) ListComments(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	list, err := h.svc.ListComments(c.Request.Context(), companyID, candidateID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load comments")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"comments": list})
}

func (h *EnterpriseHandler) CreateComment(c *gin.Context) {
	companyID, ok := h.company(c)
	if !ok {
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	var req struct {
		Body     string   `json:"body" binding:"required"`
		Mentions []string `json:"mentions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	mentions := make([]uuid.UUID, 0, len(req.Mentions))
	for _, m := range req.Mentions {
		if id, err := uuid.Parse(m); err == nil {
			mentions = append(mentions, id)
		}
	}
	var author *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		author = &uid
	}
	comment, err := h.svc.CreateComment(c.Request.Context(), companyID, candidateID, author, req.Body, mentions)
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{"comment": comment})
}

func (h *EnterpriseHandler) AssignCandidate(c *gin.Context) {
	companyID, userID, ok := h.auth(c)
	if !ok {
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	var req struct {
		AssignedTo *string `json:"assigned_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	var assignee *uuid.UUID
	if req.AssignedTo != nil && strings.TrimSpace(*req.AssignedTo) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*req.AssignedTo))
		if err != nil {
			respondError(c, http.StatusBadRequest, "validation_error", "Invalid assigned_to")
			return
		}
		assignee = &id
	}
	actor := userID
	if err := h.svc.AssignCandidate(c.Request.Context(), companyID, candidateID, &actor, assignee); err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to assign candidate")
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"assigned_to": assignee})
}

func (h *EnterpriseHandler) company(c *gin.Context) (uuid.UUID, bool) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return uuid.Nil, false
	}
	return companyID, true
}

func (h *EnterpriseHandler) auth(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	companyID, ok := h.company(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return uuid.Nil, uuid.Nil, false
	}
	return companyID, userID, true
}
