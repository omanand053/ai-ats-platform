package embedding_test

import (
	"testing"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/embedding"
)

func TestScenarioB_MissingKeyUsesLocalHash(t *testing.T) {
	p, err := embedding.NewProvider(config.EmbeddingConfig{
		Provider:     "gemini",
		Model:        "text-embedding-004",
		Version:      "v1",
		Dimensions:   384,
		GeminiAPIKey: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Model() != "local-hash" {
		t.Fatalf("expected local-hash fallback, got %s", p.Model())
	}
}

func TestScenarioA_PresentKeyUsesGemini(t *testing.T) {
	p, err := embedding.NewProvider(config.EmbeddingConfig{
		Provider:     "gemini",
		Model:        "gemini-embedding-001",
		Version:      "v1",
		Dimensions:   384,
		GeminiAPIKey: "AIzaSyDummyTestKeyOK012345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Model() != "gemini-embedding-001" {
		t.Fatalf("expected gemini-embedding-001, got %s", p.Model())
	}
}

func TestInvalidKeyFallsBackToLocal(t *testing.T) {
	p, err := embedding.NewProvider(config.EmbeddingConfig{
		Provider:     "gemini",
		GeminiAPIKey: "changeme",
		Dimensions:   384,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Model() != "local-hash" {
		t.Fatalf("got %s", p.Model())
	}
}
