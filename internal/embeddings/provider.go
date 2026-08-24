package embeddings

import (
	"context"

	"github.com/quonaro/gnostis/internal/config"
)

// Provider converts texts into embedding vectors.
type Provider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	BatchSize() int
	ModelName() string
}

// New creates a Provider from the embeddings configuration.
// Any OpenAI-compatible /v1/embeddings endpoint works (Infinity, Ollama,
// OpenAI, LocalAI, LM Studio, etc.).
func New(cfg config.Embeddings) (Provider, error) {
	return newOpenAICompatible(cfg.URL, cfg.Model, cfg.APIKey, cfg.BatchSize, cfg.MaxChars), nil
}
