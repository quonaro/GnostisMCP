package app

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/db"
	"github.com/quonaro/gnostis/internal/directory"
	"github.com/quonaro/gnostis/internal/embeddings"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/indexer"
	"github.com/quonaro/gnostis/internal/jobs"
	mcpServer "github.com/quonaro/gnostis/internal/mcp"
	"github.com/quonaro/gnostis/internal/memory"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/project"
	"github.com/quonaro/gnostis/internal/search"
	"github.com/quonaro/gnostis/internal/simhash"
	"github.com/quonaro/gnostis/internal/stats"
	"github.com/quonaro/gnostis/internal/store"
	"github.com/quonaro/gnostis/internal/symbol"
	"github.com/quonaro/gnostis/internal/watcher"
	"github.com/quonaro/gnostis/internal/web"
)

// App orchestrates configuration, indexing, search, and the MCP server.
type App struct {
	cfg            config.Config
	sqlDB          *sql.DB
	dirs           []directory.Directory
	projects       []project.Project
	store          store.VectorStore
	provider       embeddings.Provider
	engine         *search.Engine
	indexer        *indexer.Indexer
	chunker        *chunker.Chunker
	symbolIndex    *symbol.Index
	callGraph      *graph.Graph
	simhashIndex   *simhash.Index
	watcher        *watcher.Watcher
	memoryMgr      *memory.Manager
	mcp            *mcpServer.Server
	webSrv         *web.Server
	embeddingCache map[string][]float32
	progress       *progress.Progress
	indexingStats  *stats.Stats
	ProgressWriter io.Writer
	jobQueue       *jobs.Queue

	rebuildMu        sync.RWMutex
	watcherStarted   bool
	projectsSnapshot atomic.Pointer[[]project.Project]
	dirsSnapshot     atomic.Pointer[[]directory.Directory]
	modelName        atomic.Value
	watcherRestartCh chan struct{}

	changesCacheMu sync.Mutex
	changesCache   map[string]changesCacheEntry

	layoutCacheMu sync.Mutex
	layoutCache   map[string]layoutCacheEntry
}

// New builds the application from configuration.
func New(cfg config.Config) (*App, error) {
	slog.Info("initializing app", "data_dir", cfg.DataDir, "provider", cfg.Embeddings.Provider, "model", cfg.Embeddings.Model)

	sqlDB, err := db.Open(filepath.Join(cfg.DataDir, "gnostis.db"))
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	dirs, projects, err := resolveProjects(cfg, sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("resolve projects: %w", err)
	}

	ctx := context.Background()

	st, err := store.New(ctx, cfg.DataDir, sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create store: %w", err)
	}

	provider, err := embeddings.New(cfg.Embeddings)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create embeddings provider: %w", err)
	}

	engine := search.New(st, provider)

	symbolIndex, err := symbol.New(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create symbol index: %w", err)
	}

	callGraph := graph.New()
	if err := callGraph.Load(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("load call graph: %w", err)
	}

	simhashIndex, err := simhash.NewIndex(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create simhash index: %w", err)
	}

	embeddingCache := make(map[string][]float32)

	a := &App{
		cfg:              cfg,
		sqlDB:            sqlDB,
		dirs:             dirs,
		projects:         projects,
		store:            st,
		provider:         provider,
		engine:           engine,
		indexer:          indexer.New(),
		chunker:          chunker.New(),
		symbolIndex:      symbolIndex,
		callGraph:        callGraph,
		simhashIndex:     simhashIndex,
		embeddingCache:   embeddingCache,
		progress:         progress.New(sqlDB),
		indexingStats:    stats.New(sqlDB),
		jobQueue:         jobs.New(20),
		watcherRestartCh: make(chan struct{}, 1),
		changesCache:     make(map[string]changesCacheEntry),
		layoutCache:      make(map[string]layoutCacheEntry),
	}
	a.updateSnapshots(cfg, projects)

	if memoryEnabled(cfg.Memory) {
		dataDir := config.InterpolateEnv(config.DefaultMemoryDataDir)
		mgr, err := memory.NewManager(cfg.Memory, dataDir, provider, sqlDB)
		if err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("create memory manager: %w", err)
		}
		if err := mgr.Start(context.Background()); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("start memory manager: %w", err)
		}
		a.memoryMgr = mgr
	}

	mcpSrv := mcpServer.New(engine, symbolIndex, a, a.memoryMgr, projects)
	a.mcp = mcpSrv

	a.watcher = a.newWatcher()

	if cfg.Web.Enabled {
		a.webSrv = web.New(a, engine, a.mcp.StreamableHTTPHandler())
	}

	return a, nil
}

func memoryEnabled(cfg config.Memory) bool {
	return cfg.Cascade.Enabled
}

// updateSnapshots updates the lock-free copies used by status/read-only APIs.
// It must be called while rebuildMu is held (or during construction).
func (a *App) updateSnapshots(cfg config.Config, projects []project.Project) {
	snapshot := append([]project.Project(nil), projects...)
	a.projectsSnapshot.Store(&snapshot)

	dirsSnap := append([]directory.Directory(nil), a.dirs...)
	a.dirsSnapshot.Store(&dirsSnap)

	a.modelName.Store(cfg.Embeddings.Model)
}

// Run serves the MCP stdio server while performing initial indexing and
// starting the watcher in the background. The first component error stops the app.
func (a *App) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "starting app")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go a.runWatcherRestarter(ctx)
	go a.jobQueue.Start(ctx)

	errCh := make(chan error, 3)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		indexDone := make(chan error, 1)
		a.jobQueue.Submit("index", "Initial index", func(jobCtx context.Context) error {
			a.rebuildMu.Lock()
			defer a.rebuildMu.Unlock()
			a.progress.SetJobID("index")
			err := a.initialIndex(jobCtx)
			indexDone <- err
			return err
		})

		if err := <-indexDone; err != nil {
			errCh <- fmt.Errorf("initial index: %w", err)
			cancel()
			return
		}

		// If initialIndex delegated to resumeInterruptedJob, wait for that
		// job to finish before starting the watcher.
		if state, loadErr := a.progress.Load(); loadErr == nil && state.JobID != "" && state.JobID != "index" {
			a.jobQueue.Wait(state.JobID)
		}

		a.rebuildMu.Lock()
		if err := a.watcher.Start(); err != nil {
			a.rebuildMu.Unlock()
			errCh <- fmt.Errorf("start watcher: %w", err)
			cancel()
			return
		}
		a.watcherStarted = true
		a.rebuildMu.Unlock()
		<-ctx.Done()
		_ = a.watcher.Stop()
		if a.memoryMgr != nil {
			_ = a.memoryMgr.Stop()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.watchConfig(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("config watcher: %w", err)
			cancel()
		}
	}()

	if a.webSrv != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.webSrv.Start(ctx, a.cfg.Web.Port); err != nil && err != context.Canceled {
				errCh <- fmt.Errorf("web server: %w", err)
				cancel()
			}
		}()
	}

	if a.webSrv != nil {
		// Web server is enabled: run stdio in background and block on
		// OS signals so the app stays alive even when stdin is closed
		// (e.g. under a task runner like lota dev).
		go func() {
			if err := a.mcp.StartStdio(ctx); err != nil && err != context.Canceled {
				slog.WarnContext(ctx, "stdio server ended", "error", err)
			}
		}()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
	} else {
		// No web server: block on stdin as before.
		if err := a.mcp.StartStdio(ctx); err != nil {
			errCh <- fmt.Errorf("stdio server: %w", err)
		}
	}
	cancel()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

func (a *App) initialIndex(ctx context.Context) error {
	if state, err := a.progress.Load(); err != nil {
		slog.ErrorContext(ctx, "load progress", "error", err)
	} else if state.JobID != "" && (state.Status == progress.StatusRunning || state.Status == progress.StatusError) {
		if state.Status == progress.StatusError {
			slog.WarnContext(ctx, "previous indexing ended with error, resuming", "job_id", state.JobID, "error", state.Error)
		}
		return a.resumeInterruptedJob(ctx, state)
	} else if state.JobID == "" && (state.Status == progress.StatusRunning || state.Status == progress.StatusError) {
		slog.WarnContext(ctx, "previous indexing interrupted without job ID, re-indexing from scratch")
		_ = a.progress.Reset()
	}

	if state, err := a.progress.Load(); err == nil && state.Status == progress.StatusDone {
		if a.isIndexUpToDate(ctx, state.UpdatedAt) {
			slog.InfoContext(ctx, "index already up to date, skipping", "chunks", a.store.Count())
			return nil
		}
	}

	a.cleanupDeletedFiles(ctx)
	var firstErr error
	for i, dir := range a.dirs {
		slog.InfoContext(ctx, "indexing directory", "path", dir.Path, "project", a.projects[i].Name)
		if err := indexDirectoryWithRetry(ctx, a.ProgressWriter, dir, a.projects[i], a.indexer, a.chunker, a.provider, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.embeddingCache, a.progress, a.indexingStats); err != nil {
			slog.ErrorContext(ctx, "index project failed, continuing to next", "project", a.projects[i].Name, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("index %s: %w", dir.Path, err)
			}
			continue
		}
	}
	if err := a.symbolIndex.Save(); err != nil {
		slog.ErrorContext(ctx, "save symbol index", "error", err)
	}
	a.saveCallGraph()
	a.saveSimhashIndex()
	slog.InfoContext(ctx, "initial index complete", "chunks", a.store.Count())
	return firstErr
}
