package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/config"
)

// APIError is a structured Gemini HTTP failure for logging and fallback reasons.
type APIError struct {
	Status   int
	Body     string
	Model    string
	Endpoint string
}

func (e *APIError) Error() string {
	if e == nil {
		return "gemini api error"
	}
	body := strings.TrimSpace(e.Body)
	if len(body) > 500 {
		body = body[:500] + "…"
	}
	return fmt.Sprintf(
		"gemini api error status=%d model=%s endpoint=%s body=%s",
		e.Status, e.Model, e.Endpoint, body,
	)
}

// FallbackReason is a recruiter/debug-facing explanation of why Gemini failed.
func (e *APIError) FallbackReason() string {
	if e == nil {
		return "Gemini API error"
	}
	if e.Status == http.StatusNotFound {
		return config.ErrGeminiModelInvalid{Model: e.Model, Detail: compactGeminiBody(e.Body)}.Error()
	}
	detail := compactGeminiBody(e.Body)
	if detail == "" {
		detail = http.StatusText(e.Status)
	}
	return fmt.Sprintf("Gemini API error (HTTP %d, model=%s): %s", e.Status, e.Model, detail)
}

// GeminiProvider calls the Google Generative Language API.
type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	model = config.NormalizeGeminiModel(model)
	return &GeminiProvider{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GeminiProvider) Name() string  { return "gemini" }
func (p *GeminiProvider) Model() string { return p.model }

func (p *GeminiProvider) requireModel() error {
	if strings.TrimSpace(p.model) == "" {
		return config.ErrGeminiModelNotConfigured{}
	}
	return nil
}

func (p *GeminiProvider) endpointURL() string {
	return fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		p.model,
	)
}

// VerifyModel checks that the configured model exists for this API key.
func (p *GeminiProvider) VerifyModel(ctx context.Context) error {
	if err := p.requireModel(); err != nil {
		return err
	}
	if p.apiKey == "" {
		return fmt.Errorf("gemini api key is not configured")
	}
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s",
		p.model,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		apiErr := &APIError{
			Status:   resp.StatusCode,
			Body:     string(raw),
			Model:    p.model,
			Endpoint: endpoint,
		}
		logGeminiAPIError(apiErr)
		if resp.StatusCode == http.StatusNotFound {
			return config.ErrGeminiModelInvalid{Model: p.model, Detail: compactGeminiBody(string(raw))}
		}
		return apiErr
	}
	return nil
}

func (p *GeminiProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if err := p.requireModel(); err != nil {
		return nil, err
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}
	system := req.SystemPrompt
	if strings.TrimSpace(system) == "" {
		system = "You are an AI recruiting assistant. Answer using only the provided resume context."
	}
	user := req.UserPrompt
	if strings.TrimSpace(user) == "" {
		user = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", req.Context, req.Question)
	}
	prompt := system + "\n\n" + user

	endpoint := p.endpointURL()
	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		apiErr := &APIError{
			Status:   resp.StatusCode,
			Body:     string(raw),
			Model:    p.model,
			Endpoint: endpoint,
		}
		logGeminiAPIError(apiErr)
		if resp.StatusCode == http.StatusNotFound {
			return nil, config.ErrGeminiModelInvalid{Model: p.model, Detail: compactGeminiBody(string(raw))}
		}
		return nil, apiErr
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("gemini decode error model=%s endpoint=%s: %w", p.model, endpoint, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		apiErr := &APIError{
			Status:   resp.StatusCode,
			Body:     parsed.Error.Message,
			Model:    p.model,
			Endpoint: endpoint,
		}
		logGeminiAPIError(apiErr)
		if strings.Contains(strings.ToUpper(parsed.Error.Status), "NOT_FOUND") ||
			strings.Contains(strings.ToLower(parsed.Error.Message), "no longer available") {
			return nil, config.ErrGeminiModelInvalid{Model: p.model, Detail: parsed.Error.Message}
		}
		return nil, apiErr
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content model=%s endpoint=%s", p.model, endpoint)
	}
	return &GenerateResponse{
		Answer:   strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text),
		Provider: p.Name(),
		Model:    p.Model(),
	}, nil
}

func logGeminiAPIError(err *APIError) {
	if err == nil {
		return
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if env != "" && env != "development" && env != "dev" && env != "local" {
		log.Printf("🤖 Gemini API error status=%d model=%s endpoint=%s", err.Status, err.Model, err.Endpoint)
		return
	}
	log.Printf("🤖 Gemini API error (development detail)")
	log.Printf("   HTTP status: %d", err.Status)
	log.Printf("   Model name:  %s", err.Model)
	log.Printf("   Endpoint:    %s", err.Endpoint)
	log.Printf("   Response:    %s", strings.TrimSpace(err.Body))
}

func compactGeminiBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(body), &parsed) == nil && parsed.Error.Message != "" {
		msg := parsed.Error.Message
		if parsed.Error.Status != "" {
			return parsed.Error.Status + ": " + msg
		}
		return msg
	}
	if len(body) > 300 {
		return body[:300] + "…"
	}
	return body
}

// IsModelConfigError reports whether err is a Gemini model configuration problem.
func IsModelConfigError(err error) bool {
	var missing config.ErrGeminiModelNotConfigured
	var invalid config.ErrGeminiModelInvalid
	return errors.As(err, &missing) || errors.As(err, &invalid)
}
