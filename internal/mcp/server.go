package mcp

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/quonaro/gnostis/internal/coverage"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/jobs"
	"github.com/quonaro/gnostis/internal/memory"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/project"
	"github.com/quonaro/gnostis/internal/search"
	"github.com/quonaro/gnostis/internal/simhash"
	"github.com/quonaro/gnostis/internal/stats"
	"github.com/quonaro/gnostis/internal/symbol"
)

// Searcher is the subset of the search engine used by MCP tools.
type Searcher interface {
	Search(ctx context.Context, query string, filters map[string]string, topK int) ([]search.Result, error)
}

// Finder is the subset of the symbol index used by MCP tools.
type Finder interface {
	Lookup(name string) []symbol.Location
	SearchFuzzy(query string) []symbol.Location
}

// Indexer exposes the operations MCP tools can perform on the index.
type Indexer interface {
	Status() ([]string, int)
	Info() (provider, model string, symbols int)
	ProgressState() (progress.State, error)
	ProjectStats(ctx context.Context) (map[string]stats.Project, error)
	MemoryStats(ctx context.Context) []memory.ProviderStat
	MemoryProgressState() memory.ProgressState
	ProjectPath(name string) (string, error)
	ReindexFiles(ctx context.Context, paths []string) error
	StartRebuildProject(ctx context.Context, name string) (string, error)
	StartRebuildIndex(ctx context.Context) (string, error)
	AddProject(ctx context.Context, path, name string, extensions, include, exclude []string, maxFileSizeMB int) (string, error)
	EditProject(ctx context.Context, name string, extensions, include, exclude []string, maxFileSizeMB int) error
	RemoveProject(ctx context.Context, name string) error
	CheckCoverage(ctx context.Context, paths []string) []coverage.Status
	DetectChanges(ctx context.Context, project string) ([]coverage.Change, error)
	TracePath(ctx context.Context, from, to, project string, maxDepth int) (graph.TraceResult, error)
	DeadCode(ctx context.Context, project, kind string, topK int) ([]graph.DeadCodeCandidate, error)
	Architecture(ctx context.Context, project string) (*graph.Architecture, error)
	FindSimilar(ctx context.Context, path, project string, threshold float64, topK int) ([]simhash.FileMatch, error)
	GraphLayout(project string, connectedOnly bool, maxNodes int) (graph.LayoutResult, error)
	MemoryFiles(ctx context.Context) []memory.FileInfo
	Jobs() []jobs.Job
}

const serverName = "gnostis"

// version is set by the build linker to the short git commit hash.
var version string

// SetVersion sets the server version. Called from main after the linker sets main.version.
func SetVersion(v string) {
	version = v
}

// Server wraps the mcp-go server and exposes Gnostis tools.
type Server struct {
	mu            sync.RWMutex
	server        *mcpserver.MCPServer
	version       string
	engine        Searcher
	symbols       Finder
	indexer       Indexer
	memoryManager *memory.Manager
	projects      []project.Project
}

// New creates and configures the MCP server.
func New(engine Searcher, symbols Finder, indexer Indexer, memoryManager *memory.Manager, projects []project.Project) *Server {
	slog.Info("creating mcp server", "name", serverName, "version", version)
	s := &Server{
		version:       version,
		engine:        engine,
		symbols:       symbols,
		indexer:       indexer,
		memoryManager: memoryManager,
		projects:      projects,
	}

	s.server = mcpserver.NewMCPServer(
		serverName,
		version,
		mcpserver.WithToolCapabilities(false),
	)
	s.registerTools()

	return s
}

// ReloadProjects updates the project list used for path resolution and list_projects.
func (s *Server) ReloadProjects(projects []project.Project) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = projects
}

// ReloadMemoryManager replaces the memory manager used by memory tools.
func (s *Server) ReloadMemoryManager(mgr *memory.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryManager = mgr
}

// StartStdio runs the MCP server over stdio. It blocks until stdin is closed
// or SIGTERM/SIGINT is received.
func (s *Server) StartStdio(ctx context.Context) error {
	slog.InfoContext(ctx, "starting mcp stdio server", "name", serverName, "version", s.version)
	return mcpserver.ServeStdio(s.server)
}

// StreamableHTTPHandler returns an http.Handler that serves MCP over the
// Streamable HTTP transport. Multiple editors can connect to the same
// endpoint without spawning separate gnostis processes.
func (s *Server) StreamableHTTPHandler() http.Handler {
	slog.Info("creating mcp streamable http handler", "name", serverName, "version", s.version)
	srv := mcpserver.NewStreamableHTTPServer(
		s.server,
		mcpserver.WithStateful(true),
	)
	return srv
}

func (s *Server) registerTools() {
	slog.Info("registering mcp tools")
	s.server.AddTool(findSymbolTool(), mcp.NewTypedToolHandler(s.findSymbol))
	s.server.AddTool(getFileContextTool(), mcp.NewTypedToolHandler(s.getFileContext))
	s.server.AddTool(listProjectsTool(), mcp.NewTypedToolHandler(s.listProjects))
	s.server.AddTool(grepTool(), mcp.NewTypedToolHandler(s.grep))
	s.server.AddTool(listFilesTool(), mcp.NewTypedToolHandler(s.listFiles))
	s.server.AddTool(directoryTreeTool(), mcp.NewTypedToolHandler(s.directoryTree))
	s.server.AddTool(getRecentChangesTool(), mcp.NewTypedToolHandler(s.getRecentChanges))
	s.server.AddTool(reindexFilesTool(), mcp.NewTypedToolHandler(s.reindexFiles))
	s.server.AddTool(unifiedSearchTool(), mcp.NewTypedToolHandler(s.unifiedSearch))
	s.server.AddTool(getIndexStatusTool(), mcp.NewTypedToolHandler(s.getIndexStatus))
	s.server.AddTool(getIndexJobTool(), mcp.NewTypedToolHandler(s.getIndexJob))
	s.server.AddTool(rebuildProjectTool(), mcp.NewTypedToolHandler(s.rebuildProject))
	s.server.AddTool(rebuildIndexTool(), mcp.NewTypedToolHandler(s.rebuildIndex))
	s.server.AddTool(addProjectTool(), mcp.NewTypedToolHandler(s.addProject))
	s.server.AddTool(editProjectTool(), mcp.NewTypedToolHandler(s.editProject))
	s.server.AddTool(removeProjectTool(), mcp.NewTypedToolHandler(s.removeProject))
	s.server.AddTool(memorySearchTool(), mcp.NewTypedToolHandler(s.memorySearch))
	s.server.AddTool(memoryWriteTool(), mcp.NewTypedToolHandler(s.memoryWrite))
	s.server.AddTool(memoryListTool(), mcp.NewTypedToolHandler(s.memoryList))
	s.server.AddTool(memoryReadTool(), mcp.NewTypedToolHandler(s.memoryRead))
	s.server.AddTool(rebuildMemoryTool(), mcp.NewTypedToolHandler(s.rebuildMemory))
	s.server.AddTool(checkIndexCoverageTool(), mcp.NewTypedToolHandler(s.checkIndexCoverage))
	s.server.AddTool(detectChangesTool(), mcp.NewTypedToolHandler(s.detectChanges))
	s.server.AddTool(tracePathTool(), mcp.NewTypedToolHandler(s.tracePath))
	s.server.AddTool(deadCodeTool(), mcp.NewTypedToolHandler(s.deadCode))
	s.server.AddTool(getArchitectureTool(), mcp.NewTypedToolHandler(s.getArchitecture))
	s.server.AddTool(findSimilarTool(), mcp.NewTypedToolHandler(s.findSimilar))
	s.server.AddTool(graphLayoutTool(), mcp.NewTypedToolHandler(s.graphLayout))
	s.server.AddTool(memoryFilesTool(), mcp.NewTypedToolHandler(s.memoryFiles))
}

func findSymbolTool() mcp.Tool {
	return mcp.NewTool("find_symbol",
		mcp.WithDescription("Find the definition of a named symbol"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Symbol name")),
		mcp.WithString("project", mcp.Description("Project name")),
		mcp.WithString("language", mcp.Description("Language filter")),
	)
}

func getFileContextTool() mcp.Tool {
	return mcp.NewTool("get_file_context",
		mcp.WithDescription("Read a file or a range of lines"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute file path")),
		mcp.WithNumber("start_line", mcp.Description("First line (1-based)")),
		mcp.WithNumber("end_line", mcp.Description("Last line (1-based)")),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all indexed projects"),
	)
}
