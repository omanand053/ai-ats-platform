package llm

import "context"

// GenerateRequest is the input to an LLM provider for RAG answering.
type GenerateRequest struct {
	SystemPrompt string
	UserPrompt   string
	Question     string
	Context      string
}

// GenerateResponse is the model output.
type GenerateResponse struct {
	Answer   string
	Provider string
	Model    string
}

// Provider generates text answers. Swap Gemini/OpenAI/mock without changing RAG business logic.
type Provider interface {
	Name() string
	Model() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
