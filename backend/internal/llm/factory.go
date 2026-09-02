package llm

import (
	"log"
	"strings"

	"ai-ats-platform/backend/internal/config"
)

// NewProvider constructs the LLM provider.
// Default config is gemini; selection is driven by GEMINI_API_KEY so developers
// do not need to edit LLM_PROVIDER to switch between gemini and mock.
// Never returns an error for a missing/invalid API key.
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = "gemini"
	}
	model := config.NormalizeGeminiModel(cfg.Model)

	switch provider {
	case "mock", "local", "placeholder":
		log.Printf("LLM Provider: mock")
		return NewMockProvider(firstNonEmpty(model, "mock-rag-v1")), nil

	case "openai":
		if strings.TrimSpace(cfg.OpenAIAPIKey) == "" {
			log.Printf("LLM Provider: mock (OPENAI_API_KEY missing; falling back)")
			return NewMockProvider(firstNonEmpty(model, "mock-rag-v1")), nil
		}
		p := NewOpenAIProvider(cfg.OpenAIAPIKey, firstNonEmpty(model, "gpt-4o-mini"))
		log.Printf("LLM Provider: openai model=%s", p.Model())
		return p, nil

	case "gemini", "google":
		status := config.ResolveGeminiAPIKey(cfg.GeminiAPIKey)
		if status != config.GeminiKeyOK {
			log.Printf("LLM Provider: mock (GEMINI_API_KEY %s; falling back)", status)
			return NewMockProvider(firstNonEmpty(model, "mock-rag-v1")), nil
		}
		if model == "" {
			log.Printf("LLM Provider: mock (GEMINI_MODEL not configured)")
			return NewMockProvider("mock-rag-v1"), nil
		}
		p := NewGeminiProvider(cfg.GeminiAPIKey, model)
		log.Printf("LLM Provider: gemini model=%s", p.Model())
		return p, nil

	default:
		status := config.ResolveGeminiAPIKey(cfg.GeminiAPIKey)
		if status == config.GeminiKeyOK {
			if model == "" {
				log.Printf("LLM Provider: mock (GEMINI_MODEL not configured)")
				return NewMockProvider("mock-rag-v1"), nil
			}
			p := NewGeminiProvider(cfg.GeminiAPIKey, model)
			log.Printf("LLM Provider: gemini model=%s", p.Model())
			return p, nil
		}
		log.Printf("LLM Provider: mock (unsupported %q; GEMINI_API_KEY %s)", cfg.Provider, status)
		return NewMockProvider(firstNonEmpty(model, "mock-rag-v1")), nil
	}
}

// EnsureProvider never returns nil; falls back to mock if provider is nil.
func EnsureProvider(p Provider) Provider {
	if p == nil {
		log.Printf("LLM Provider: mock (nil provider)")
		return NewMockProvider("mock-rag-v1")
	}
	return p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
