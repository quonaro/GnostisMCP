package app

import (
	"context"
	"fmt"
	"os"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/indexer"
	"github.com/quonaro/gnostis/internal/simhash"
)

// simhashAddChunks adds simhash fingerprints for chunks to the index.
func simhashAddChunks(idx *simhash.Index, chunks []chunker.Chunk) {
	for _, c := range chunks {
		idx.Add(c.Simhash, simhash.Meta{
			ProjectID: c.ProjectID,
			Path:      c.Path,
			Symbol:    c.Symbol,
			StartLine: c.StartLine,
			EndLine:   c.EndLine,
		})
	}
}

// FindSimilar finds near-duplicate code blocks for a file.
func (a *App) FindSimilar(_ context.Context, path, projectName string, threshold float64, topK int) ([]simhash.FileMatch, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if threshold <= 0 {
		threshold = 0.85
	}
	if topK <= 0 {
		topK = 5
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path must be a file, not a directory")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	projectID := ""
	if projectName != "" {
		pp, err := a.ProjectPath(projectName)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", projectName, err)
		}
		projectID = pp
	}

	fileInfo := indexer.FileInfo{
		ProjectID: projectID,
		Path:      path,
		Content:   string(content),
	}

	ch := chunker.New()
	chunks, err := ch.ChunkFile(context.Background(), fileInfo)
	if err != nil {
		return nil, fmt.Errorf("chunk file: %w", err)
	}

	var results []simhash.FileMatch
	for _, c := range chunks {
		matches := a.simhashIndex.FindSimilar(c.Simhash, threshold, path, topK)
		if len(matches) > 0 {
			results = append(results, simhash.FileMatch{
				Symbol:    c.Symbol,
				StartLine: c.StartLine,
				Matches:   matches,
			})
		}
	}

	return results, nil
}
