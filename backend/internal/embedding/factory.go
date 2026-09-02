package embedding

import (
	"log"
	"strings"

	"ai-ats-platform/backend/internal/config"
)

// NewProvider constructs the embedding provider.
// Default config is gemini; selection is driven by GEMINI_API_KEY so developers
// do not need to edit EMBEDDING_PROVIDER to switch between gemini and local-hash.
// Never returns an error for a missing/invalid API key.
func NewProvider(cfg config.EmbeddingConfig) (Provider, error) {
	dims := cfg.Dimensions
	if dims <= 0 {
		dims = 384
	}
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = "v1"
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = "gemini"
	}

	// Explicit local override (rare; defaults stay gemini).
	if provider == "local" || provider == "local-hash" {
		log.Printf("Embedding Provider: local-hash")
		return NewLocalHashProvider("local-hash", version, dims), nil
	}

	// gemini / google / unknown → key-driven selection (default path).
	status := config.ResolveGeminiAPIKey(cfg.GeminiAPIKey)
	if status != config.GeminiKeyOK {
		log.Printf("Embedding Provider: local-hash (GEMINI_API_KEY %s; falling back)", status)
		return NewLocalHashProvider("local-hash", version, dims), nil
	}

	model := firstNonEmpty(cfg.Model, defaultGeminiEmbeddingModel)
	log.Printf("Embedding Provider: gemini")
	return NewGeminiProvider(cfg.GeminiAPIKey, model, version, dims), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
