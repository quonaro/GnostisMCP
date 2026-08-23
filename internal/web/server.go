package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/quonaro/gnostis/internal/coverage"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/jobs"
	"github.com/quonaro/gnostis/internal/memory"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/search"
	"github.com/quonaro/gnostis/internal/stats"
	"github.com/quonaro/gnostis/internal/sysmetrics"
)

// App is the subset of the application used by the web server.
type App interface {
	Status() ([]string, int)
	Info() (provider, model string, symbols int)
	ProgressState() (progress.State, error)
	ProjectStats(ctx context.Context) (map[string]stats.Project, error)
	MemoryStats(ctx context.Context) []memory.ProviderStat
	MemoryProgressState() memory.ProgressState
	MemoryFiles(ctx context.Context) []memory.FileInfo
	MemoryDataDir() string
	StartRebuildProject(ctx context.Context, name string) (string, error)
	StartRebuildIndex(ctx context.Context) (string, error)
	AddProject(ctx context.Context, path, name string, extensions, include, exclude []string, maxFileSizeMB int) (string, error)
	EditProject(ctx context.Context, name string, extensions, include, exclude []string, maxFileSizeMB int) error
	RemoveProject(ctx context.Context, name string) error
	ProjectPath(name string) (string, error)
	ReindexFiles(ctx context.Context, paths []string) error
	GraphLayout(project string, connectedOnly bool, maxNodes int) (graph.LayoutResult, error)
	Architecture(ctx context.Context, project string) (*graph.Architecture, error)
	DeadCode(ctx context.Context, project, kind string, topK int) ([]graph.DeadCodeCandidate, error)
	DetectChanges(ctx context.Context, project string) ([]coverage.Change, error)
	Jobs() []jobs.Job
}

// Searcher is the subset of the search engine used by the web server.
type Searcher interface {
	Search(ctx context.Context, query string, filters map[string]string, topK int) ([]search.Result, error)
}

// Server is the HTTP dashboard server.
type Server struct {
	app        App
	search     Searcher
	mcpHandler http.Handler
	metrics    *sysmetrics.Collector
	mux        *http.ServeMux
}

// New creates a new web server. If mcpHandler is non-nil it is mounted at /mcp
// to serve MCP clients over the Streamable HTTP transport.
func New(app App, srch Searcher, mcpHandler http.Handler) *Server {
	s := &Server{
		app:        app,
		search:     srch,
		mcpHandler: mcpHandler,
		metrics:    sysmetrics.NewCollector(),
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("POST /api/rebuild/project", s.handleRebuildProject)
	s.mux.HandleFunc("POST /api/rebuild/index", s.handleRebuildIndex)
	s.mux.HandleFunc("GET /api/projects/pick-directory", s.handlePickDirectory)
	s.mux.HandleFunc("POST /api/projects/add", s.handleAddProject)
	s.mux.HandleFunc("POST /api/projects/edit", s.handleEditProject)
	s.mux.HandleFunc("POST /api/projects/remove", s.handleRemoveProject)
	s.mux.HandleFunc("POST /api/projects/open", s.handleOpenProject)
	s.mux.HandleFunc("POST /api/reindex", s.handleReindex)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/graph", s.handleGraph)
	s.mux.HandleFunc("GET /api/architecture", s.handleArchitecture)
	s.mux.HandleFunc("GET /api/dead-code", s.handleDeadCode)
	s.mux.HandleFunc("GET /api/changes", s.handleChanges)
	s.mux.HandleFunc("GET /api/memory/files", s.handleMemoryFiles)
	s.mux.HandleFunc("POST /api/memory/open", s.handleOpenMemoryFile)
	if s.mcpHandler != nil {
		s.mux.Handle("/mcp", s.mcpHandler)
	}
	s.mux.Handle("/", s.handleSPA())
}

// Start launches the HTTP server. It blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutting down web server")
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	slog.InfoContext(ctx, "starting web server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "error", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
