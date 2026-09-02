package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"strings"
)

// LocalHashProvider is a deterministic, offline embedding provider used until a
// production model (e.g. OpenAI) is configured. Same text always yields the same vector.
type LocalHashProvider struct {
	model      string
	version    string
	dimensions int
}

func NewLocalHashProvider(model, version string, dimensions int) *LocalHashProvider {
	if dimensions <= 0 {
		dimensions = 384
	}
	if strings.TrimSpace(model) == "" {
		model = "local-hash"
	}
	if strings.TrimSpace(version) == "" {
		version = "v1"
	}
	return &LocalHashProvider{model: model, version: version, dimensions: dimensions}
}

func (p *LocalHashProvider) Model() string    { return p.model }
func (p *LocalHashProvider) Version() string  { return p.version }
func (p *LocalHashProvider) Dimensions() int  { return p.dimensions }

func (p *LocalHashProvider) Embed(_ context.Context, text string) ([]float32, error) {
	normalized := normalizeText(text)
	vec := make([]float32, p.dimensions)
	if normalized == "" {
		return vec, nil
	}

	// Project token hashes into a unit-normalized bag-of-hashes vector.
	tokens := strings.Fields(normalized)
	if len(tokens) == 0 {
		tokens = []string{normalized}
	}
	for _, tok := range tokens {
		h := sha256.Sum256([]byte(tok))
		for i := 0; i < 8; i++ {
			idx := int(binary.BigEndian.Uint32(h[i*4:(i+1)*4]) % uint32(p.dimensions))
			sign := float32(1)
			if h[i]&1 == 1 {
				sign = -1
			}
			vec[idx] += sign
		}
		// Second pass with salted token for denser coverage.
		h2 := sha256.Sum256([]byte(p.model + ":" + tok))
		for i := 0; i < 8; i++ {
			idx := int(binary.BigEndian.Uint32(h2[i*4:(i+1)*4]) % uint32(p.dimensions))
			sign := float32(1)
			if h2[i]&1 == 1 {
				sign = -1
			}
			vec[idx] += sign * 0.5
		}
	}

	var sumSquares float64
	for _, v := range vec {
		sumSquares += float64(v) * float64(v)
	}
	norm := math.Sqrt(sumSquares)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}
	return vec, nil
}

func normalizeText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(text))), " ")
}

// ContentHash returns a stable hash for skip-if-unchanged checks.
func ContentHash(text, model, version string) string {
	sum := sha256.Sum256([]byte(model + "\x00" + version + "\x00" + normalizeText(text)))
	return hexEncode(sum[:])
}

func hexEncode(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}
