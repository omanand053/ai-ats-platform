package embedding

import "context"

// Provider generates vector embeddings for text. Swap implementations without
// changing business logic that depends on EmbeddingService.
type Provider interface {
	// Embed returns a fixed-length vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
	// Model is the provider model identifier stored with each embedding.
	Model() string
	// Version is the embedding schema/version string stored with each embedding.
	Version() string
	// Dimensions is the vector length (must match DB vector(N)).
	Dimensions() int
}
