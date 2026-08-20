package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/search"
	"github.com/quonaro/gnostis/internal/stats"
)

type mockApp struct {
	projects    []string
	chunks      int
	provider    string
	model       string
	symbols     int
	pstate      progress.State
	pstats      map[string]stats.Project
	jobID       string
	err         error
	addedName   string
	removedName string
}

func (m *mockApp) Status() ([]string, int)                { return m.projects, m.chunks }
func (m *mockApp) Info() (string, string, int)            { return m.provider, m.model, m.symbols }
func (m *mockApp) ProgressState() (progress.State, error) { return m.pstate, nil }
func (m *mockApp) ProjectStats(_ context.Context) (map[string]stats.Project, error) {
	return m.pstats, nil
}
func (m *mockApp) StartRebuildProject(_ context.Context, name string) (string, error) {
	return m.jobID, m.err
}
func (m *mockApp) StartRebuildIndex(_ context.Context) (string, error) {
	return m.jobID, m.err
}
func (m *mockApp) AddProject(_ context.Context, path, name string, _ []string, _ []string, _ []string, _ int) (string, error) {
	m.addedName = name
	if name == "" {
		return "auto-name", m.err
	}
	return name, m.err
}
func (m *mockApp) EditProject(_ context.Context, name string, _ []string, _ []string, _ []string, _ int) error {
	return m.err
}
func (m *mockApp) RemoveProject(_ context.Context, name string) error {
	m.removedName = name
	return m.err
}
func (m *mockApp) ReindexFiles(_ context.Context, _ []string) error { return m.err }

type mockSearcher struct {
	results []search.Result
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ string, _ map[string]string, _ int) ([]search.Result, error) {
	return m.results, m.err
}

func newTestServer() *Server {
	app := &mockApp{
		projects: []string{"proj1", "proj2"},
		chunks:   42,
		provider: "ollama",
		model:    "nomic-embed-text",
		symbols:  100,
		pstate:   progress.State{Status: progress.StatusIdle},
		pstats: map[string]stats.Project{
			"proj1": {Chunks: 20},
			"proj2": {Chunks: 22},
		},
		jobID: "test-job-1",
	}
	srch := &mockSearcher{
		results: []search.Result{
			{ID: "r1", Path: "/foo.go", Symbol: "Foo", StartLine: 1, EndLine: 10, Score: 0.95},
		},
	}
	return New(app, srch, nil)
}

func TestHandleStatus(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	s.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp statusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TotalChunks != 42 {
		t.Errorf("expected 42 chunks, got %d", resp.TotalChunks)
	}
	if resp.Provider != "ollama" {
		t.Errorf("expected ollama, got %s", resp.Provider)
	}
	if len(resp.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(resp.Projects))
	}
}

func TestHandleSearch(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=foo&top_k=5", nil)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected 1 result, got %v", resp["count"])
	}
}

func TestHandleSearchMissingQuery(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRebuildProject(t *testing.T) {
	s := newTestServer()
	body := `{"name":"proj1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rebuild/project", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRebuildProject(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	var resp jobResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.JobID != "test-job-1" {
		t.Errorf("expected test-job-1, got %s", resp.JobID)
	}
}

func TestHandleRebuildProjectMissingName(t *testing.T) {
	s := newTestServer()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/rebuild/project", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRebuildProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAddProject(t *testing.T) {
	s := newTestServer()
	body := `{"path":"/tmp/proj","name":"newproj"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/add", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleAddProject(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp nameResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "newproj" {
		t.Errorf("expected newproj, got %s", resp.Name)
	}
}

func TestHandleRemoveProject(t *testing.T) {
	s := newTestServer()
	body := `{"name":"proj1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/remove", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRemoveProject(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleReindex(t *testing.T) {
	s := newTestServer()
	body := `{"paths":["/tmp/file.go"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleReindex(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleReindexEmptyPaths(t *testing.T) {
	s := newTestServer()
	body := `{"paths":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleReindex(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
