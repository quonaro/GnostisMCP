package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/quonaro/gnostis/internal/simhash"
)

type mockSimhashIndexer struct {
	mockIndexer
	matches []simhash.FileMatch
	err     error
}

func (m *mockSimhashIndexer) FindSimilar(_ context.Context, path, project string, threshold float64, topK int) ([]simhash.FileMatch, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.matches, nil
}

func TestFindSimilar_MissingPath(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockSimhashIndexer{}, nil, nil)
	res, err := srv.findSimilar(context.Background(), mcp.CallToolRequest{}, findSimilarArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing path")
	}
	assertTextEquals(t, res, "path is required")
}

func TestFindSimilar_IndexerNotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil, nil)
	res, err := srv.findSimilar(context.Background(), mcp.CallToolRequest{}, findSimilarArgs{Path: "/tmp/test.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for nil indexer")
	}
	assertTextEquals(t, res, "not configured")
}

func TestFindSimilar_NoMatches(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockSimhashIndexer{}, nil, nil)
	res, err := srv.findSimilar(context.Background(), mcp.CallToolRequest{}, findSimilarArgs{Path: "/tmp/test.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTextEquals(t, res, "[]")
}

func TestFindSimilar_WithMatches(t *testing.T) {
	matches := []simhash.FileMatch{
		{
			Symbol:    "process",
			StartLine: 10,
			Matches: []simhash.Match{
				{
					Meta: simhash.Meta{
						Path:      "/other/file.go",
						Symbol:    "process",
						StartLine: 5,
						EndLine:   20,
					},
					Similarity: 0.95,
				},
			},
		},
	}
	srv := New(&mockSearcher{}, nil, &mockSimhashIndexer{matches: matches}, nil, nil)
	res, err := srv.findSimilar(context.Background(), mcp.CallToolRequest{}, findSimilarArgs{Path: "/tmp/test.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatal("expected success result")
	}
	assertTextEquals(t, res, "process")
	assertTextEquals(t, res, "/other/file.go")
	assertTextEquals(t, res, "0.95")
}

func TestFindSimilar_WithProjectFile(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := New(&mockSearcher{}, nil, &mockSimhashIndexer{}, nil, nil)
	res, err := srv.findSimilar(context.Background(), mcp.CallToolRequest{}, findSimilarArgs{Path: testFile, Project: "test", Threshold: 0.9, TopK: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTextEquals(t, res, "[]")
}
