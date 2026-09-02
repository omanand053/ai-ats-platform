package llm_test

import (
	"context"
	"strings"
	"testing"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/llm"
)

func TestMockProviderGenerate(t *testing.T) {
	p := llm.NewMockProvider("mock-rag-v1")
	out, err := p.Generate(context.Background(), llm.GenerateRequest{
		Question: "Who knows Go?",
		Context:  "Candidate Ada: Go, PostgreSQL",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Answer, "MOCK LLM PLACEHOLDER") {
		t.Fatalf("expected mock placeholder, got %q", out.Answer)
	}
	if out.Provider != "mock" {
		t.Fatalf("provider=%s", out.Provider)
	}
}

func TestScenarioB_MissingKeyUsesMock(t *testing.T) {
	cases := []config.LLMConfig{
		{Provider: "gemini", GeminiAPIKey: ""},
		{Provider: "", GeminiAPIKey: ""},
		{Provider: "gemini", GeminiAPIKey: "changeme"},
		{Provider: "openai", OpenAIAPIKey: ""},
		{Provider: "unknown-xyz"},
	}
	for _, cfg := range cases {
		p, err := llm.NewProvider(cfg)
		if err != nil {
			t.Fatalf("cfg=%+v err=%v", cfg, err)
		}
		if p.Name() != "mock" {
			t.Fatalf("cfg=%+v expected mock, got %s", cfg, p.Name())
		}
	}
}

func TestScenarioA_PresentKeyUsesGemini(t *testing.T) {
	p, err := llm.NewProvider(config.LLMConfig{
		Provider:     "gemini",
		Model:        "gemini-2.5-flash-lite",
		GeminiAPIKey: "AIzaSyDummyTestKeyOK012345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "gemini" {
		t.Fatalf("expected gemini, got %s", p.Name())
	}
	if p.Model() != "gemini-2.5-flash-lite" {
		t.Fatalf("expected configured model, got %s", p.Model())
	}
}

func TestScenarioA_PresentKeyWithoutModelUsesMock(t *testing.T) {
	p, err := llm.NewProvider(config.LLMConfig{
		Provider:     "gemini",
		GeminiAPIKey: "AIzaSyDummyTestKeyOK012345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "mock" {
		t.Fatalf("expected mock when GEMINI_MODEL missing, got %s", p.Name())
	}
}
