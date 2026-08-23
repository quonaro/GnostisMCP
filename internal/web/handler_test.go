package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quonaro/gnostis/internal/coverage"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/jobs"
	"github.com/quonaro/gnostis/internal/memory"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/search"
	"github.com/quonaro/gnostis/internal/stats"
)

type mockApp struct {
	projects     []string
	chunks       int
	provider     string
	model        string
	symbols      int
	pstate       progress.State
	pstats       map[string]stats.Project
	jobID        string
	err          error
	addedName    string
	removedName  string
	layoutResult graph.LayoutResult
	layoutErr    error
	arch         *graph.Architecture
	deadCode     []graph.DeadCodeCandidate
	changes      []coverage.Change
}

func (m *mockApp) Status() ([]string, int)                { return m.projects, m.chunks }
func (m *mockApp) Info() (string, string, int)            { return m.provider, m.model, m.symbols }
func (m *mockApp) ProgressState() (progress.State, error) { return m.pstate, nil }
func (m *mockApp) ProjectStats(_ context.Context) (map[string]stats.Project, error) {
	return m.pstats, nil
}
func (m *mockApp) MemoryStats(_ context.Context) []memory.ProviderStat {
	return nil
}
func (m *mockApp) MemoryProgressState() memory.ProgressState {
	return memory.ProgressState{Status: memory.MemStatusIdle}
}
func (m *mockApp) MemoryFiles(_ context.Context) []memory.FileInfo {
	return nil
}
func (m *mockApp) MemoryDataDir() string {
	return "/tmp/memory"
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
func (m *mockApp) ProjectPath(name string) (string, error) {
	return "/tmp/" + name, nil
}
func (m *mockApp) ReindexFiles(_ context.Context, _ []string) error { return m.err }
func (m *mockApp) GraphLayout(_ string, _ bool, _ int) (graph.LayoutResult, error) {
	return m.layoutResult, m.layoutErr
}
func (m *mockApp) Architecture(_ context.Context, _ string) (*graph.Architecture, error) {
	return m.arch, m.err
}
func (m *mockApp) DeadCode(_ context.Context, _, _ string, _ int) ([]graph.DeadCodeCandidate, error) {
	return m.deadCode, m.err
}
func (m *mockApp) DetectChanges(_ context.Context, _ string) ([]coverage.Change, error) {
	return m.changes, m.err
}
func (m *mockApp) Jobs() []jobs.Job { return nil }

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
			"proj1": {Chunks: 20, Model: "nomic-embed-text"},
			"proj2": {Chunks: 22, Model: "nomic-embed-text"},
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

func TestHandleMemoryFiles(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/memory/files", nil)
	w := httptest.NewRecorder()

	s.handleMemoryFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestHandleOpenMemoryFileMissingPath(t *testing.T) {
	s := newTestServer()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/memory/open", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleOpenMemoryFile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleOpenMemoryFileOutsideDir(t *testing.T) {
	s := newTestServer()
	body := `{"path":"/etc/passwd"}`
	req := httptest.NewRequest(http.MethodPost, "/api/memory/open", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleOpenMemoryFile(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleGraph(t *testing.T) {
	s := newTestServer()
	m := s.app.(*mockApp)
	m.layoutResult = graph.LayoutResult{
		Nodes: []graph.LayoutNode{
			{Node: graph.Node{ID: "n1", Path: "/foo.go", Symbol: "Foo", Kind: "function"}, X: 1.5, Y: 2.5, Degree: 3},
		},
		Edges:      []graph.ResolvedEdge{{From: "n1", To: "n2"}},
		TotalNodes: 1,
		TotalEdges: 1,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/graph?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp graphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Nodes) != 1 || resp.Nodes[0].Symbol != "Foo" {
		t.Errorf("unexpected nodes: %+v", resp.Nodes)
	}
	if resp.Nodes[0].X != 1.5 || resp.Nodes[0].Y != 2.5 || resp.Nodes[0].Degree != 3 {
		t.Errorf("unexpected node layout: %+v", resp.Nodes[0])
	}
	if len(resp.Edges) != 1 || resp.Edges[0].To != "n2" {
		t.Errorf("unexpected edges: %+v", resp.Edges)
	}
}

func TestHandleGraphEmptyProject(t *testing.T) {
	s := newTestServer()
	m := s.app.(*mockApp)
	m.layoutResult = graph.LayoutResult{
		Nodes: []graph.LayoutNode{
			{Node: graph.Node{ID: "n1", Path: "/foo.go", Symbol: "Foo"}, X: 0, Y: 0, Degree: 0},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	w := httptest.NewRecorder()

	s.handleGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleArchitecture(t *testing.T) {
	s := newTestServer()
	m := s.app.(*mockApp)
	m.arch = &graph.Architecture{
		Project:      "proj1",
		TotalFiles:   10,
		TotalSymbols: 50,
		TotalEdges:   100,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/architecture?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleArchitecture(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp graph.Architecture
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Project != "proj1" || resp.TotalSymbols != 50 {
		t.Errorf("unexpected architecture: %+v", resp)
	}
}

func TestHandleArchitectureMissingProject(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/architecture", nil)
	w := httptest.NewRecorder()

	s.handleArchitecture(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDeadCode(t *testing.T) {
	s := newTestServer()
	m := s.app.(*mockApp)
	m.deadCode = []graph.DeadCodeCandidate{
		{Symbol: "unusedFunc", Path: "/foo.go", Kind: "function"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/dead-code?project=proj1&kind=function&top_k=10", nil)
	w := httptest.NewRecorder()

	s.handleDeadCode(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

func TestHandleDeadCodeMissingProject(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/dead-code", nil)
	w := httptest.NewRecorder()

	s.handleDeadCode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleChanges(t *testing.T) {
	s := newTestServer()
	m := s.app.(*mockApp)
	m.changes = []coverage.Change{
		{Path: "/foo.go", Status: "modified"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/changes?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleChanges(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

func TestHandleChangesMissingProject(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/changes", nil)
	w := httptest.NewRecorder()

	s.handleChanges(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGraphNilProducesEmptyArrays(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/graph?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleGraph(w, req)

	body := w.Body.String()
	if strings.Contains(body, "null") {
		t.Errorf("graph response should not contain null: %s", body)
	}
	var resp graphResponse
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Nodes == nil || resp.Edges == nil {
		t.Errorf("nodes and edges should be empty slices, not nil: %+v", resp)
	}
}

func TestHandleDeadCodeNilProducesEmptyArray(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/dead-code?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleDeadCode(w, req)

	body := w.Body.String()
	if strings.Contains(body, "null") {
		t.Errorf("dead-code response should not contain null: %s", body)
	}
}

func TestHandleChangesNilProducesEmptyArray(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/changes?project=proj1", nil)
	w := httptest.NewRecorder()

	s.handleChanges(w, req)

	body := w.Body.String()
	if strings.Contains(body, "null") {
		t.Errorf("changes response should not contain null: %s", body)
	}
}
