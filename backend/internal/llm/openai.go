package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider calls the OpenAI Chat Completions API.
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string  { return "openai" }
func (p *OpenAIProvider) Model() string { return p.model }

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai api key is not configured")
	}
	system := req.SystemPrompt
	if strings.TrimSpace(system) == "" {
		system = "You are an AI recruiting assistant. Answer using only the provided resume context."
	}
	user := req.UserPrompt
	if strings.TrimSpace(user) == "" {
		user = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", req.Context, req.Question)
	}

	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.2,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai api error: %s", strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}
	return &GenerateResponse{
		Answer:   strings.TrimSpace(parsed.Choices[0].Message.Content),
		Provider: p.Name(),
		Model:    p.Model(),
	}, nil
}
