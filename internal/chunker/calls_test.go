package chunker

import (
	"context"
	"testing"

	"github.com/quonaro/gnostis/internal/indexer"
)

func TestChunkGoCalls(t *testing.T) {
	src := `package main

import "fmt"

func greet(name string) {
	fmt.Println("hello", name)
}

func main() {
	greet("world")
}
`
	file := indexer.FileInfo{
		Path:      "test.go",
		Content:   src,
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

	var mainChunk *Chunk
	for i := range chunks {
		if chunks[i].Symbol == "main" {
			mainChunk = &chunks[i]
			break
		}
	}
	if mainChunk == nil {
		t.Fatal("no main chunk found")
	}
	if mainChunk.Kind != "function" {
		t.Errorf("expected kind 'function', got %q", mainChunk.Kind)
	}
	if len(mainChunk.Calls) != 1 {
		t.Fatalf("expected 1 call in main, got %d", len(mainChunk.Calls))
	}
	if mainChunk.Calls[0].Name != "greet" {
		t.Errorf("expected call to 'greet', got %q", mainChunk.Calls[0].Name)
	}
}

func TestChunkGoMethodCalls(t *testing.T) {
	src := `package main

type Server struct{}

func (s *Server) Start() {
	s.run()
}

func (s *Server) run() {}
`
	file := indexer.FileInfo{
		Path:      "test.go",
		Content:   src,
		ProjectID: "test",
		Hash:      "abc",
	}

	c := New()
	chunks, err := c.ChunkFile(context.Background(), file)
	if err != nil {
		t.Fatalf("ChunkFile: %v", err)
	}

	var startChunk *Chunk
	for i := range chunks {
		if chunks[i].Symbol == "Start" {
			startChunk = &chunks[i]
			break
		}
	}
	if startChunk == nil {
		t.Fatal("no Start chunk found")
	}
	if startChunk.Kind != "method" {
		t.Errorf("expected kind 'method', got %q", startChunk.Kind)
	}
	if len(startChunk.Calls) != 1 {
		t.Fatalf("expected 1 call in Start, got %d", len(startChunk.Calls))
	}
	if startChunk.Calls[0].Name != "run" {
		t.Errorf("expected call to 'run', got %q", startChunk.Calls[0].Name)
	}
}

func TestChunkPythonCalls(t *testing.T) {
	src := `def main():
    print("hello")
    greet("world")

def greet(name):
    pass
`
	file := indexer.FileInfo{
		Path:      "test.py",
		Content:   src,
		ProjectID: "test",
		Hash:      "abc",
	}

	c := New()
	chunks, err := c.ChunkFile(context.Background(), file)
	if err != nil {
		t.Fatalf("ChunkFile: %v", err)
	}

	var mainChunk *Chunk
	for i := range chunks {
		if chunks[i].Symbol == "main" {
			mainChunk = &chunks[i]
			break
		}
	}
	if mainChunk == nil {
		t.Fatal("no main chunk found")
	}
	if mainChunk.Kind != "function" {
		t.Errorf("expected kind 'function', got %q", mainChunk.Kind)
	}
	if len(mainChunk.Calls) != 2 {
		t.Fatalf("expected 2 calls in main, got %d", len(mainChunk.Calls))
	}
}

func TestChunkMarkdownNoCalls(t *testing.T) {
	src := `# Title

Some content here.

## Section

More content.
`
	file := indexer.FileInfo{
		Path:      "test.md",
		Content:   src,
		ProjectID: "test",
		Hash:      "abc",
	}

	c := New()
	chunks, err := c.ChunkFile(context.Background(), file)
	if err != nil {
		t.Fatalf("ChunkFile: %v", err)
	}
	for _, ch := range chunks {
		if ch.Kind != "document" {
			t.Errorf("expected kind 'document', got %q", ch.Kind)
		}
		if len(ch.Calls) != 0 {
			t.Errorf("expected no calls in markdown chunk, got %d", len(ch.Calls))
		}
	}
}
