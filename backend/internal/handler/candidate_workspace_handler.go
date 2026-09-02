package handler

import (
	"errors"
	"net/http"
	"strings"

	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type noteRequest struct {
	Body string `json:"body" binding:"required,min=1,max=4000"`
}

func (h *CandidateHandler) ListNotes(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}
	if h.workspaceService == nil {
		respondError(c, http.StatusServiceUnavailable, "workspace_unavailable", "Recruiter workspace is not configured")
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	notes, err := h.workspaceService.ListNotes(c.Request.Context(), companyID, candidateID)
	if err != nil {
		handleWorkspaceError(c, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"notes": notes})
}

func (h *CandidateHandler) CreateNote(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}
	if h.workspaceService == nil {
		respondError(c, http.StatusServiceUnavailable, "workspace_unavailable", "Recruiter workspace is not configured")
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	var author *uuid.UUID
	if userID, ok := middleware.GetUserID(c); ok {
		author = &userID
	}
	note, err := h.workspaceService.CreateNote(c.Request.Context(), companyID, candidateID, author, req.Body)
	if err != nil {
		handleWorkspaceError(c, err)
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{"note": note})
}

func (h *CandidateHandler) DeleteNote(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}
	if h.workspaceService == nil {
		respondError(c, http.StatusServiceUnavailable, "workspace_unavailable", "Recruiter workspace is not configured")
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_note_id", "Invalid note ID")
		return
	}
	if err := h.workspaceService.DeleteNote(c.Request.Context(), companyID, candidateID, noteID); err != nil {
		handleWorkspaceError(c, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "Note deleted"})
}

func (h *CandidateHandler) Timeline(c *gin.Context) {
	companyID, ok := h.companyContext(c)
	if !ok {
		return
	}
	if h.workspaceService == nil {
		respondError(c, http.StatusServiceUnavailable, "workspace_unavailable", "Recruiter workspace is not configured")
		return
	}
	candidateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid candidate ID")
		return
	}
	items, err := h.workspaceService.Timeline(c.Request.Context(), companyID, candidateID)
	if err != nil {
		handleWorkspaceError(c, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"timeline": items})
}

func handleWorkspaceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCandidateNotFound):
		respondError(c, http.StatusNotFound, "candidate_not_found", "Candidate not found")
	case errors.Is(err, service.ErrCandidateNoteNotFound):
		respondError(c, http.StatusNotFound, "note_not_found", "Note not found")
	default:
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "too long") {
			respondError(c, http.StatusBadRequest, "validation_error", msg)
			return
		}
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
