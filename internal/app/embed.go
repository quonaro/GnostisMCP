package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/embeddings"
	"github.com/quonaro/gnostis/internal/symbol"
	"golang.org/x/sync/errgroup"
)

func chunksToSymbolChunks(chunks []chunker.Chunk) []symbol.Chunk {
	out := make([]symbol.Chunk, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, symbol.Chunk{
			ProjectID: c.ProjectID,
			Path:      c.Path,
			Language:  c.Language,
			Symbol:    c.Symbol,
			Signature: c.Signature,
			StartLine: c.StartLine,
			EndLine:   c.EndLine,
		})
	}
	return out
}

func embedChunks(ctx context.Context, provider embeddings.Provider, chunks []chunker.Chunk, cache map[string][]float32, onEmbedded func(int)) ([][]float32, error) {
	results := make([][]float32, len(chunks))
	var missingIndices []int
	var missingTexts []string

	for i, c := range chunks {
		if cache == nil {
			missingIndices = append(missingIndices, i)
			missingTexts = append(missingTexts, c.Content)
			continue
		}
		if v, ok := cache[c.ID]; ok {
			results[i] = v
			continue
		}
		missingIndices = append(missingIndices, i)
		missingTexts = append(missingTexts, c.Content)
	}

	if len(missingTexts) > 0 {
		batchSize := provider.BatchSize()
		if batchSize <= 0 {
			batchSize = 32
		}

		totalBatches := (len(missingTexts) + batchSize - 1) / batchSize
		slog.DebugContext(ctx, "embedding chunks", "count", len(missingTexts), "cached", len(chunks)-len(missingTexts), "batch_size", batchSize, "batches", totalBatches, "model", provider.ModelName())

		const maxConcurrent = 4
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(maxConcurrent)

		var cacheMu sync.Mutex
		var progressMu sync.Mutex

		for i := 0; i < len(missingTexts); i += batchSize {
			end := i + batchSize
			if end > len(missingTexts) {
				end = len(missingTexts)
			}

			start, stop := i, end
			g.Go(func() error {
				vectors, err := provider.Embed(gctx, missingTexts[start:stop])
				if err != nil {
					return fmt.Errorf("embed batch %d-%d: %w", start, stop, err)
				}
				if len(vectors) != stop-start {
					return fmt.Errorf("expected %d embeddings, got %d", stop-start, len(vectors))
				}

				for j, idx := range missingIndices[start:stop] {
					results[idx] = vectors[j]
					if cache != nil {
						cacheMu.Lock()
						cache[chunks[idx].ID] = vectors[j]
						cacheMu.Unlock()
					}
				}

				if onEmbedded != nil {
					progressMu.Lock()
					onEmbedded(len(vectors))
					progressMu.Unlock()
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	return results, nil
}
