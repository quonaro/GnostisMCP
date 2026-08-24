package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// openAICompatible is an HTTP client for OpenAI-compatible /v1/embeddings endpoints.
type openAICompatible struct {
	client    *http.Client
	url       string
	model     string
	apiKey    string
	batchSize int
	maxChars  int
}

func newOpenAICompatible(url, model, apiKey string, batchSize, maxChars int) *openAICompatible {
	if batchSize <= 0 {
		batchSize = 32
	}
	if maxChars <= 0 {
		maxChars = 8000
	}
	return &openAICompatible{
		client:    &http.Client{Timeout: 120 * time.Second},
		url:       url,
		model:     model,
		apiKey:    apiKey,
		batchSize: batchSize,
		maxChars:  maxChars,
	}
}

func (p *openAICompatible) ModelName() string {
	return p.model
}

func (p *openAICompatible) BatchSize() int {
	return p.batchSize
}

func (p *openAICompatible) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	slog.InfoContext(ctx, "embedding", "count", len(texts), "model", p.model)
	var all [][]float32

	for i := 0; i < len(texts); i += p.batchSize {
		end := i + p.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch, err := p.embedBatch(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("embed batch %d-%d: %w", i, end, err)
		}
		all = append(all, batch...)
	}

	return all, nil
}

func (p *openAICompatible) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	truncated := make([]string, len(texts))
	for i, t := range texts {
		if len(t) > p.maxChars {
			slog.WarnContext(ctx, "truncating embedding input", "original_len", len(t), "max_chars", p.maxChars)
			t = t[:p.maxChars]
		}
		truncated[i] = t
	}

	body := map[string]any{
		"model": p.model,
		"input": truncated,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := truncateString(string(respBody), 512)
		slog.ErrorContext(ctx, "embedding failed", "status", resp.StatusCode, "body", preview)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, preview)
	}

	var parsed embeddingsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	out := make([][]float32, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		vec := make([]float32, len(item.Embedding))
		for i, v := range item.Embedding {
			vec[i] = float32(v)
		}
		out = append(out, vec)
	}

	if len(out) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(out))
	}

	dims := 0
	if len(out) > 0 {
		dims = len(out[0])
	}
	slog.InfoContext(ctx, "embedded", "count", len(out), "dims", dims, "body", truncateString(string(respBody), 32))

	return out, nil
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}
