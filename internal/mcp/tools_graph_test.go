package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/graph"
)

type mockGraphIndexer struct {
	mockIndexer
	traceResult graph.TraceResult
	traceErr    error
	deadCode    []graph.DeadCodeCandidate
	deadCodeErr error
}

func (m *mockGraphIndexer) TracePath(_ context.Context, from, to, project string, maxDepth int) (graph.TraceResult, error) {
	if m.traceErr != nil {
		return graph.TraceResult{}, m.traceErr
	}
	return m.traceResult, nil
}

func (m *mockGraphIndexer) DeadCode(_ context.Context, project, kind string, topK int) ([]graph.DeadCodeCandidate, error) {
	if m.deadCodeErr != nil {
		return nil, m.deadCodeErr
	}
	return m.deadCode, nil
}

func TestTracePath_MissingFrom(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{To: "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing from")
	}
}

func TestTracePath_MissingTo(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{From: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing to")
	}
}

func TestTracePath_NotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{From: "foo", To: "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when indexer not configured")
	}
}

func TestTracePath_Found(t *testing.T) {
	mock := &mockGraphIndexer{
		traceResult: graph.TraceResult{
			Found: true,
			Depth: 2,
			Hops: []graph.TraceHop{
				{Symbol: "main", Path: "/foo.go", Kind: "function"},
				{Symbol: "helper", Path: "/bar.go", Kind: "function"},
				{Symbol: "db", Path: "/baz.go", Kind: "function"},
			},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{From: "main", To: "db"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}

	var got graph.TraceResult
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.Found || len(got.Hops) != 3 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestTracePath_NotFound(t *testing.T) {
	mock := &mockGraphIndexer{
		traceResult: graph.TraceResult{Found: false},
	}
	srv := New(&mockSearcher{}, nil, mock, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{From: "a", To: "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got graph.TraceResult
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Found {
		t.Error("expected Found=false")
	}
}

func TestTracePath_SymbolNotFound(t *testing.T) {
	mock := &mockGraphIndexer{
		traceErr: errors.New(`symbol "x" not found in call graph`),
	}
	srv := New(&mockSearcher{}, nil, mock, nil, nil)
	res, err := srv.tracePath(context.Background(), mcp.CallToolRequest{}, tracePathArgs{From: "x", To: "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown symbol")
	}
}

func TestDeadCode_MissingProject(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil, nil)
	res, err := srv.deadCode(context.Background(), mcp.CallToolRequest{}, deadCodeArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing project")
	}
}

func TestDeadCode_NotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil, nil)
	res, err := srv.deadCode(context.Background(), mcp.CallToolRequest{}, deadCodeArgs{Project: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when indexer not configured")
	}
}

func TestDeadCode_Results(t *testing.T) {
	mock := &mockGraphIndexer{
		deadCode: []graph.DeadCodeCandidate{
			{Symbol: "unused", Path: "/foo.go", Kind: "function"},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil, nil)
	res, err := srv.deadCode(context.Background(), mcp.CallToolRequest{}, deadCodeArgs{Project: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}

	var got []graph.DeadCodeCandidate
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].Symbol != "unused" {
		t.Errorf("unexpected candidates: %+v", got)
	}
}

func TestDeadCode_EmptyResults(t *testing.T) {
	mock := &mockGraphIndexer{}
	srv := New(&mockSearcher{}, nil, mock, nil, nil)
	res, err := srv.deadCode(context.Background(), mcp.CallToolRequest{}, deadCodeArgs{Project: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []graph.DeadCodeCandidate
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}
