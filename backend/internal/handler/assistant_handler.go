package handler

import (
	"net/http"
	"strings"

	"ai-ats-platform/backend/internal/assistant"
	"ai-ats-platform/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// AssistantHandler serves POST /assistant/chat with intent routing:
// ATS_DATA → SQL services | RAG → embeddings + Gemini | GENERAL → Gemini.
// Job-scoped RAG at /jobs/:id/ai-assistant is unchanged.
type AssistantHandler struct {
	router *assistant.Router
}

func NewAssistantHandler(router *assistant.Router) *AssistantHandler {
	return &AssistantHandler{router: router}
}

// Chat handles POST /assistant/chat
// Body: { "prompt": string, "resume_id"?: string }
func (h *AssistantHandler) Chat(c *gin.Context) {
	var req struct {
		Prompt   string `json:"prompt"`
		ResumeID string `json:"resume_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "Invalid request body. Expected { prompt: string }.")
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		respondError(c, http.StatusBadRequest, "validation_error", "prompt is required")
		return
	}

	companyID, ok := middleware.GetCompanyID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}

	if h.router == nil {
		respondError(c, http.StatusServiceUnavailable, "assistant_unavailable", "Assistant not configured")
		return
	}

	resp, err := h.router.Ask(c.Request.Context(), assistant.AskRequest{
		CompanyID: companyID,
		Query:     prompt,
		ResumeID:  strings.TrimSpace(req.ResumeID),
	})
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "invalid resume_id"):
			respondError(c, http.StatusBadRequest, "validation_error", msg)
		case strings.Contains(msg, "resume not found"):
			respondError(c, http.StatusNotFound, "not_found", msg)
		case strings.Contains(msg, "unavailable"):
			respondError(c, http.StatusServiceUnavailable, "assistant_unavailable", msg)
		default:
			respondError(c, http.StatusBadGateway, "assistant_error", msg)
		}
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"answer":                   resp.Answer,
		"reply":                    resp.Answer,
		"confidence":               resp.Confidence,
		"source":                   resp.Source,
		"intent":                   resp.Intent,
		"suggested_actions":        resp.SuggestedActions,
		"source_documents":         resp.SourceDocuments,
		"retrieved_context_count":  resp.RetrievedChunks,
		"provider":                 resp.Provider,
		"model":                    resp.Model,
		"data":                     resp.Data,
	})
}
