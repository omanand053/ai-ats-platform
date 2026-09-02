package llm

import (
	"context"
	"fmt"
	"strings"
)

// MockProvider keeps the RAG pipeline functional without a real API key.
type MockProvider struct {
	model string
}

func NewMockProvider(model string) *MockProvider {
	if strings.TrimSpace(model) == "" {
		model = "mock-rag-v1"
	}
	return &MockProvider{model: model}
}

func (p *MockProvider) Name() string  { return "mock" }
func (p *MockProvider) Model() string { return p.model }

func (p *MockProvider) Generate(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		question = strings.TrimSpace(req.UserPrompt)
	}
	contextBlock := strings.TrimSpace(req.Context)
	excerpt := contextBlock
	if len(excerpt) > 600 {
		excerpt = excerpt[:600] + "…"
	}
	if excerpt == "" {
		excerpt = "(no resume context retrieved)"
	}

	answer := fmt.Sprintf(
		"[MOCK LLM PLACEHOLDER — configure LLM_PROVIDER=openai|gemini with an API key for real answers]\n\n"+
			"Question: %s\n\n"+
			"Based only on the retrieved resume context below, here is a structured placeholder answer:\n"+
			"- The assistant would cite only candidates present in the retrieved context.\n"+
			"- No external knowledge would be invented beyond that context.\n\n"+
			"Retrieved context excerpt:\n%s",
		question,
		excerpt,
	)

	return &GenerateResponse{
		Answer:   answer,
		Provider: p.Name(),
		Model:    p.Model(),
	}, nil
}
