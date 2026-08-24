package chunker

import (
	"context"
	"strings"
	"testing"

	"github.com/quonaro/gnostis/internal/indexer"
)

func TestSplitLargeChunks(t *testing.T) {
	// Create a markdown file with no headers — would produce a single wholeFileChunk.
	longLine := strings.Repeat("a", 500)
	content := strings.Repeat(longLine+"\n", 400) // ~200K chars, exceeds maxChunkChars

	file := indexer.FileInfo{
		Path:      "big.md",
		Content:   content,
		ProjectID: "test",
		Hash:      "abc",
	}

	c := New()
	chunks, err := c.ChunkFile(context.Background(), file)
	if err != nil {
		t.Fatalf("ChunkFile: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	for i, ch := range chunks {
		if len(ch.Content) > maxChunkChars {
			t.Errorf("chunk %d exceeds maxChunkChars: %d chars", i, len(ch.Content))
		}
		if ch.Content == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

func TestSplitLargeChunksPreservesSmallChunks(t *testing.T) {
	chunks := []Chunk{
		{Content: "small chunk", Symbol: "foo"},
		{Content: "another small", Symbol: "bar"},
	}
	result := splitLargeChunks(chunks)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
}
