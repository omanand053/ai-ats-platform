package handler

import (
	"net/http"
	"strings"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/llm"

	"github.com/gin-gonic/gin"
)

type LLMHandler struct {
	geminiKey   string
	geminiModel string
}

func NewLLMHandler(geminiKey, geminiModel string) *LLMHandler {
	return &LLMHandler{
		geminiKey:   strings.TrimSpace(geminiKey),
		geminiModel: config.NormalizeGeminiModel(geminiModel),
	}
}

func (h *LLMHandler) GeminiFallback(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
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

	if h.geminiKey == "" {
		respondError(c, http.StatusInternalServerError, "gemini_unavailable", "Gemini fallback is not configured.")
		return
	}
	if h.geminiModel == "" {
		respondError(c, http.StatusInternalServerError, "gemini_model_not_configured", config.ErrGeminiModelNotConfigured{}.Error())
		return
	}

	provider := llm.NewGeminiProvider(h.geminiKey, h.geminiModel)

	gen, err := provider.Generate(c.Request.Context(), llm.GenerateRequest{
		SystemPrompt: "You are an AI assistant for an Applicant Tracking System. Answer the user's prompt directly and clearly.",
		UserPrompt:   prompt,
		Question:     prompt,
		Context:      "",
	})
	if err != nil {
		if llm.IsModelConfigError(err) {
			respondError(c, http.StatusBadRequest, "gemini_model_invalid", err.Error())
			return
		}
		respondError(c, http.StatusBadGateway, "gemini_error", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"reply": gen.Answer})
}
