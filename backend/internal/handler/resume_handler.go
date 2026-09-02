package handler

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ResumeHandler struct {
	resumeService *service.ResumeService
}

func NewResumeHandler(resumeService *service.ResumeService) *ResumeHandler {
	return &ResumeHandler{resumeService: resumeService}
}

func (h *ResumeHandler) Upload(c *gin.Context) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "Resume file is required (field name: file)")
		return
	}

	if fileHeader.Size > service.MaxResumeBytes {
		respondError(c, http.StatusBadRequest, "file_too_large", "Resume must be 5MB or smaller")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "Unable to read uploaded file")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, service.MaxResumeBytes+1))
	if err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "Unable to read uploaded file")
		return
	}
	if int64(len(data)) > service.MaxResumeBytes {
		respondError(c, http.StatusBadRequest, "file_too_large", "Resume must be 5MB or smaller")
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	result, err := h.resumeService.UploadAndParse(c.Request.Context(), companyID, userID, fileHeader.Filename, mimeType, data)
	if err != nil {
		handleResumeError(c, err)
		return
	}

	respondSuccess(c, http.StatusCreated, result)
}

func (h *ResumeHandler) Download(c *gin.Context) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}

	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid resume ID")
		return
	}

	resume, data, err := h.resumeService.GetFile(c.Request.Context(), resumeID, companyID)
	if err != nil {
		handleResumeError(c, err)
		return
	}

	contentType := resume.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	disposition := "inline"
	if c.Query("download") == "1" || strings.EqualFold(c.Query("disposition"), "attachment") {
		disposition = "attachment"
	}
	safeName := strings.ReplaceAll(filepath.Base(resume.FileName), `"`, "")
	safeName = strings.ReplaceAll(safeName, "\n", "")
	safeName = strings.ReplaceAll(safeName, "\r", "")
	if safeName == "" || safeName == "." || safeName == ".." {
		safeName = "resume"
	}
	c.Header("Content-Disposition", disposition+`; filename="`+safeName+`"`)
	c.Data(http.StatusOK, contentType, data)
}

type attachResumeRequest struct {
	CandidateID string `json:"candidate_id" binding:"required"`
}

func (h *ResumeHandler) Attach(c *gin.Context) {
	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}

	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "Invalid resume ID")
		return
	}

	var req attachResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	candidateID, err := uuid.Parse(req.CandidateID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_candidate_id", "Invalid candidate ID")
		return
	}

	if err := h.resumeService.AttachCandidate(c.Request.Context(), resumeID, candidateID, companyID); err != nil {
		handleResumeError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Resume linked to candidate"})
}

func handleResumeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidResumeFile):
		respondError(c, http.StatusBadRequest, "invalid_file", "Only PDF and DOCX files up to 5MB are allowed")
	case errors.Is(err, service.ErrResumeTooLarge):
		respondError(c, http.StatusBadRequest, "file_too_large", "Resume must be 5MB or smaller")
	case errors.Is(err, service.ErrResumeNotFound):
		respondError(c, http.StatusNotFound, "resume_not_found", "Resume not found")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
