package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/graph"
)

type mockArchIndexer struct {
	mockIndexer
	arch    *graph.Architecture
	archErr error
}

func (m *mockArchIndexer) Architecture(_ context.Context, project string) (*graph.Architecture, error) {
	if m.archErr != nil {
		return nil, m.archErr
	}
	return m.arch, nil
}

func TestGetArchitecture_MissingProject(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)
	res, err := srv.getArchitecture(context.Background(), mcp.CallToolRequest{}, getArchitectureArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing project")
	}
}

func TestGetArchitecture_NotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil)
	res, err := srv.getArchitecture(context.Background(), mcp.CallToolRequest{}, getArchitectureArgs{Project: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when indexer not configured")
	}
}

func TestGetArchitecture_Success(t *testing.T) {
	mock := &mockArchIndexer{
		arch: &graph.Architecture{
			Project:    "test",
			TotalFiles: 10,
			Languages:  map[string]int{"go": 8, "python": 2},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.getArchitecture(context.Background(), mcp.CallToolRequest{}, getArchitectureArgs{Project: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}

	var got graph.Architecture
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Project != "test" || got.TotalFiles != 10 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestGetArchitecture_ProjectNotFound(t *testing.T) {
	mock := &mockArchIndexer{
		archErr: errors.New(`project "unknown" not found`),
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.getArchitecture(context.Background(), mcp.CallToolRequest{}, getArchitectureArgs{Project: "unknown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown project")
	}
}
