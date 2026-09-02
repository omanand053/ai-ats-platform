package config

import (
	"fmt"
	"strings"
)

// GeminiAPIKeyStatus describes whether GEMINI_API_KEY can drive Gemini providers.
type GeminiAPIKeyStatus string

const (
	GeminiKeyOK      GeminiAPIKeyStatus = "ok"
	GeminiKeyMissing GeminiAPIKeyStatus = "missing"
	GeminiKeyInvalid GeminiAPIKeyStatus = "invalid"
)

// ResolveGeminiAPIKey classifies a Gemini API key without calling the network.
// Missing / empty / placeholder / too-short keys are not usable.
func ResolveGeminiAPIKey(key string) GeminiAPIKeyStatus {
	k := strings.TrimSpace(key)
	if k == "" {
		return GeminiKeyMissing
	}
	lower := strings.ToLower(k)
	placeholders := []string{
		"your-api-key",
		"your_api_key",
		"changeme",
		"change-me",
		"replace-me",
		"replace_me",
		"todo",
		"xxx",
		"gemini_api_key",
		"paste-key-here",
		"paste_key_here",
		"<gemini_api_key>",
		"none",
		"null",
	}
	for _, p := range placeholders {
		if lower == p {
			return GeminiKeyInvalid
		}
	}
	if strings.Contains(lower, "your-") || strings.Contains(lower, "your_") {
		return GeminiKeyInvalid
	}
	if len(k) < 16 {
		return GeminiKeyInvalid
	}
	return GeminiKeyOK
}

// HasUsableGeminiAPIKey reports whether Gemini providers should be selected.
func HasUsableGeminiAPIKey(key string) bool {
	return ResolveGeminiAPIKey(key) == GeminiKeyOK
}

// NormalizeGeminiModel strips the optional "models/" prefix and trims space.
func NormalizeGeminiModel(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	return strings.TrimSpace(model)
}

// ErrGeminiModelNotConfigured is returned when GEMINI_MODEL / LLM_MODEL is empty.
type ErrGeminiModelNotConfigured struct{}

func (e ErrGeminiModelNotConfigured) Error() string {
	return "GEMINI_MODEL is not configured. Set GEMINI_MODEL in .env to a supported Gemini model (see https://ai.google.dev/gemini-api/docs/models)."
}

// ErrGeminiModelInvalid is returned when the configured model is rejected by Google.
type ErrGeminiModelInvalid struct {
	Model  string
	Detail string
}

func (e ErrGeminiModelInvalid) Error() string {
	model := strings.TrimSpace(e.Model)
	if model == "" {
		model = "(empty)"
	}
	detail := strings.TrimSpace(e.Detail)
	if detail == "" {
		return fmt.Sprintf(
			"configured Gemini model %q is invalid or unavailable. Update GEMINI_MODEL in .env to a supported model.",
			model,
		)
	}
	return fmt.Sprintf(
		"configured Gemini model %q is invalid or unavailable (%s). Update GEMINI_MODEL in .env to a supported model.",
		model, detail,
	)
}

// RequireGeminiModel returns a normalized model or ErrGeminiModelNotConfigured.
func RequireGeminiModel(model string) (string, error) {
	m := NormalizeGeminiModel(model)
	if m == "" {
		return "", ErrGeminiModelNotConfigured{}
	}
	return m, nil
}
