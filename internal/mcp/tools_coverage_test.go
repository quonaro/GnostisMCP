package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/coverage"
)

type mockCoverageIndexer struct {
	mockIndexer
	statuses []coverage.Status
	changes  []coverage.Change
	err      error
}

func (m *mockCoverageIndexer) CheckCoverage(_ context.Context, paths []string) []coverage.Status {
	return m.statuses
}

func (m *mockCoverageIndexer) DetectChanges(_ context.Context, project string) ([]coverage.Change, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.changes, nil
}

func TestCheckIndexCoverage_NoPaths(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)
	res, err := srv.checkIndexCoverage(context.Background(), mcp.CallToolRequest{}, checkCoverageArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for empty paths")
	}
	assertTextEquals(t, res, "paths is required")
}

func TestCheckIndexCoverage_NotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil)
	res, err := srv.checkIndexCoverage(context.Background(), mcp.CallToolRequest{}, checkCoverageArgs{Paths: []string{"/foo"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when indexer is not configured")
	}
}

func TestCheckIndexCoverage_Indexed(t *testing.T) {
	mock := &mockCoverageIndexer{
		statuses: []coverage.Status{
			{Path: "/foo/bar.go", Status: "indexed"},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.checkIndexCoverage(context.Background(), mcp.CallToolRequest{}, checkCoverageArgs{Paths: []string{"/foo/bar.go"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}

	var got []coverage.Status
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal statuses: %v", err)
	}
	if len(got) != 1 || got[0].Status != "indexed" {
		t.Errorf("unexpected statuses: %+v", got)
	}
}

func TestCheckIndexCoverage_Stale(t *testing.T) {
	mock := &mockCoverageIndexer{
		statuses: []coverage.Status{
			{Path: "/foo/bar.go", Status: "stale"},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.checkIndexCoverage(context.Background(), mcp.CallToolRequest{}, checkCoverageArgs{Paths: []string{"/foo/bar.go"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []coverage.Status
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal statuses: %v", err)
	}
	if len(got) != 1 || got[0].Status != "stale" {
		t.Errorf("unexpected statuses: %+v", got)
	}
}

func TestDetectChanges_NoProject(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)
	res, err := srv.detectChanges(context.Background(), mcp.CallToolRequest{}, detectChangesArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for empty project")
	}
	assertTextEquals(t, res, "project is required")
}

func TestDetectChanges_NotConfigured(t *testing.T) {
	srv := New(&mockSearcher{}, nil, nil, nil)
	res, err := srv.detectChanges(context.Background(), mcp.CallToolRequest{}, detectChangesArgs{Project: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when indexer is not configured")
	}
}

func TestDetectChanges_OK(t *testing.T) {
	mock := &mockCoverageIndexer{
		changes: []coverage.Change{
			{Path: "/foo/bar.go", Status: "modified"},
			{Path: "/foo/baz.go", Status: "new"},
		},
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.detectChanges(context.Background(), mcp.CallToolRequest{}, detectChangesArgs{Project: "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}

	var got []coverage.Change
	if err := json.Unmarshal([]byte(extractText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal changes: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(got))
	}
	if got[0].Status != "modified" || got[1].Status != "new" {
		t.Errorf("unexpected changes: %+v", got)
	}
}

func TestDetectChanges_UnknownProject(t *testing.T) {
	mock := &mockCoverageIndexer{
		err: errors.New(`project "nope" not found`),
	}
	srv := New(&mockSearcher{}, nil, mock, nil)
	res, err := srv.detectChanges(context.Background(), mcp.CallToolRequest{}, detectChangesArgs{Project: "nope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown project")
	}
}
