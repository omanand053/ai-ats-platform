package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/config"
)

const (
	defaultGeminiEmbeddingModel = "gemini-embedding-001"
	defaultGeminiEmbedVersion   = "v1"
	geminiEmbedMaxChars         = 8000 // stay within model token limits
)

// GeminiProvider calls Google's Generative Language embedContent API.
type GeminiProvider struct {
	apiKey     string
	model      string
	version    string
	dimensions int
	client     *http.Client
}

func NewGeminiProvider(apiKey, model, version string, dimensions int) *GeminiProvider {
	if strings.TrimSpace(model) == "" {
		model = defaultGeminiEmbeddingModel
	}
	if strings.TrimSpace(version) == "" {
		version = defaultGeminiEmbedVersion
	}
	if dimensions <= 0 {
		dimensions = 384
	}
	return &GeminiProvider{
		apiKey:     strings.TrimSpace(apiKey),
		model:      strings.TrimSpace(model),
		version:    strings.TrimSpace(version),
		dimensions: dimensions,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GeminiProvider) Model() string   { return p.model }
func (p *GeminiProvider) Version() string { return p.version }
func (p *GeminiProvider) Dimensions() int { return p.dimensions }

type geminiEmbedRequest struct {
	Model                string               `json:"model"`
	Content              geminiEmbedContent   `json:"content"`
	TaskType             string               `json:"taskType,omitempty"`
	OutputDimensionality int                  `json:"outputDimensionality,omitempty"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedPart struct {
	Text string `json:"text"`
}

type geminiEmbedResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (p *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return make([]float32, p.dimensions), nil
	}
	if len(text) > geminiEmbedMaxChars {
		text = text[:geminiEmbedMaxChars]
	}

	modelName := strings.TrimPrefix(p.model, "models/")
	modelPath := "models/" + modelName
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent",
		url.PathEscape(modelName),
	)

	body := geminiEmbedRequest{
		Model: modelPath,
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
		// Symmetric comparison for job↔resume semantic search.
		TaskType:             "SEMANTIC_SIMILARITY",
		OutputDimensionality: p.dimensions,
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

	var parsed geminiEmbedResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("gemini embed decode: %w (body=%s)", err, truncateForErr(string(raw), 300))
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return nil, fmt.Errorf("gemini embed api error (%d): %s", resp.StatusCode, msg)
	}
	if len(parsed.Embedding.Values) == 0 {
		return nil, fmt.Errorf("gemini embed returned empty vector")
	}

	vec := make([]float32, len(parsed.Embedding.Values))
	for i, v := range parsed.Embedding.Values {
		vec[i] = float32(v)
	}

	// Keep storage at vector(384): truncate if API ignored outputDimensionality.
	if len(vec) > p.dimensions {
		vec = vec[:p.dimensions]
	}
	if len(vec) != p.dimensions {
		return nil, fmt.Errorf("gemini embed dimension mismatch: got %d want %d", len(vec), p.dimensions)
	}

	// Truncated embeddings should be L2-normalized for cosine distance.
	l2Normalize(vec)
	return vec, nil
}

// ProbeAPIKey performs a tiny embed request to verify the key and model work
// with the Generative Language embedContent API. Returns nil when usable.
func ProbeAPIKey(ctx context.Context, apiKey, model string, dimensions int) error {
	if config.ResolveGeminiAPIKey(apiKey) != config.GeminiKeyOK {
		return fmt.Errorf("gemini api key %s", config.ResolveGeminiAPIKey(apiKey))
	}
	if strings.TrimSpace(model) == "" {
		model = defaultGeminiEmbeddingModel
	}
	if dimensions <= 0 {
		dimensions = 384
	}
	p := NewGeminiProvider(apiKey, model, "v1", dimensions)
	_, err := p.Embed(ctx, "ping")
	return err
}

func l2Normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return
	}
	for i := range vec {
		vec[i] = float32(float64(vec[i]) / norm)
	}
}

func truncateForErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
