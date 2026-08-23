package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/quonaro/gnostis/internal/progress"
)

// RebuildProject removes the existing index for a single project and reindexes it.
func (a *App) RebuildProject(ctx context.Context, name string) error {
	for i, p := range a.projects {
		if p.Name != name {
			continue
		}

		slog.InfoContext(ctx, "rebuilding project", "project", p.Name, "path", a.dirs[i].Path)

		if err := a.deleteChunksByPrefix(ctx, a.dirs[i].Path); err != nil {
			_ = a.progress.Fail(err)
			return fmt.Errorf("delete project chunks: %w", err)
		}

		if err := indexDirectoryWithRetry(ctx, a.ProgressWriter, a.dirs[i], p, a.indexer, a.chunker, a.provider, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.embeddingCache, a.progress, a.indexingStats); err != nil {
			return fmt.Errorf("index %s: %w", a.dirs[i].Path, err)
		}

		if err := a.symbolIndex.Save(); err != nil {
			_ = a.progress.Fail(err)
			return fmt.Errorf("save symbol index: %w", err)
		}
		a.saveCallGraph()
		a.saveSimhashIndex()

		a.InvalidateChangesCache(p.Name)
		a.InvalidateLayoutCache(p.Name)
		slog.InfoContext(ctx, "project rebuild complete", "project", p.Name, "chunks", a.store.Count())
		return nil
	}

	return fmt.Errorf("project %q not found", name)
}

// StartRebuildProject starts a rebuild job for a single project and returns a job ID.
func (a *App) StartRebuildProject(ctx context.Context, name string) (string, error) {
	desc := fmt.Sprintf("Reindex project: %s", name)
	id := a.jobQueue.Submit("project:"+name, desc, func(jobCtx context.Context) error {
		a.rebuildMu.Lock()
		defer a.rebuildMu.Unlock()
		a.progress.SetJobID("project:" + name)
		return a.RebuildProject(jobCtx, name)
	})
	return id, nil
}

// RebuildIndex removes the existing index and rebuilds everything.
func (a *App) RebuildIndex(ctx context.Context) error {
	wasStarted := a.watcherStarted
	if a.watcher != nil && wasStarted {
		if err := a.watcher.Stop(); err != nil {
			return fmt.Errorf("stop watcher: %w", err)
		}
	}
	defer func() {
		if a.watcher != nil && wasStarted {
			if err := a.watcher.Start(); err != nil {
				slog.ErrorContext(ctx, "restart watcher", "error", err)
			}
		}
	}()

	if err := a.deleteChunksByPrefix(ctx, ""); err != nil {
		_ = a.progress.Fail(err)
		return fmt.Errorf("delete all chunks: %w", err)
	}

	if err := a.initialIndex(ctx); err != nil {
		_ = a.progress.Fail(err)
		return fmt.Errorf("initial index: %w", err)
	}

	a.invalidateAllChangesCaches()
	a.invalidateAllLayoutCaches()
	slog.InfoContext(ctx, "full rebuild complete", "chunks", a.store.Count())
	return nil
}

// StartRebuildIndex starts a full rebuild job and returns a job ID.
func (a *App) StartRebuildIndex(ctx context.Context) (string, error) {
	desc := "Reindex all projects"
	id := a.jobQueue.Submit("index", desc, func(jobCtx context.Context) error {
		a.rebuildMu.Lock()
		defer a.rebuildMu.Unlock()
		a.progress.SetJobID("index")
		return a.RebuildIndex(jobCtx)
	})
	return id, nil
}

// resumeInterruptedJob restarts a rebuild that was in progress or failed when
// the previous process exited. It expects progress.Status to be Running or Error.
// Unlike RebuildProject/RebuildIndex, resume methods do NOT delete existing
// chunks — the hash check in chunkFilesParallel skips already-processed files.
func (a *App) resumeInterruptedJob(ctx context.Context, state progress.State) error {
	slog.InfoContext(ctx, "resuming interrupted rebuild", "job_id", state.JobID, "project", state.Project)

	prefix := jobPrefix(state.JobID)
	switch {
	case strings.HasPrefix(prefix, "project:"):
		name := strings.TrimPrefix(prefix, "project:")
		if name == "" {
			_ = a.progress.Fail(fmt.Errorf("interrupted by restart: empty project name"))
			return nil
		}
		a.jobQueue.SubmitWithID(state.JobID, prefix, fmt.Sprintf("Resume project: %s", name), func(jobCtx context.Context) error {
			a.rebuildMu.Lock()
			defer a.rebuildMu.Unlock()
			a.progress.SetJobID(state.JobID)
			return a.resumeRebuildProject(jobCtx, name)
		})
		return nil
	case prefix == "index":
		a.jobQueue.SubmitWithID(state.JobID, prefix, "Resume full rebuild", func(jobCtx context.Context) error {
			a.rebuildMu.Lock()
			defer a.rebuildMu.Unlock()
			a.progress.SetJobID(state.JobID)
			return a.resumeRebuildIndex(jobCtx)
		})
		return nil
	default:
		slog.WarnContext(ctx, "unknown running job, marking as failed", "job_id", state.JobID)
		_ = a.progress.Fail(fmt.Errorf("interrupted by restart: unknown job type"))
		return nil
	}
}

// resumeRebuildProject reindexes a single project without deleting existing
// chunks. The hash check in chunkFilesParallel skips files that were already
// processed before the interruption.
func (a *App) resumeRebuildProject(ctx context.Context, name string) error {
	for i, p := range a.projects {
		if p.Name != name {
			continue
		}

		slog.InfoContext(ctx, "resuming project rebuild", "project", p.Name, "path", a.dirs[i].Path)

		if err := indexDirectoryWithRetry(ctx, a.ProgressWriter, a.dirs[i], p, a.indexer, a.chunker, a.provider, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.embeddingCache, a.progress, a.indexingStats); err != nil {
			return fmt.Errorf("index %s: %w", a.dirs[i].Path, err)
		}

		if err := a.symbolIndex.Save(); err != nil {
			_ = a.progress.Fail(err)
			return fmt.Errorf("save symbol index: %w", err)
		}
		a.saveCallGraph()
		a.saveSimhashIndex()

		a.InvalidateChangesCache(p.Name)
		a.InvalidateLayoutCache(p.Name)
		slog.InfoContext(ctx, "project resume complete", "project", p.Name, "chunks", a.store.Count())
		return nil
	}

	return fmt.Errorf("project %q not found", name)
}

// resumeRebuildIndex reindexes all projects without deleting existing chunks.
// The hash check in chunkFilesParallel skips files that were already processed.
func (a *App) resumeRebuildIndex(ctx context.Context) error {
	slog.InfoContext(ctx, "resuming full index rebuild")

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
	a.invalidateAllChangesCaches()
	a.invalidateAllLayoutCaches()
	slog.InfoContext(ctx, "index resume complete", "chunks", a.store.Count())
	return firstErr
}

// jobPrefix extracts the logical prefix from a job ID such as
// "project:RuobrOld-1783920062303548722" or "index-1783920062303548722".
func jobPrefix(id string) string {
	if idx := strings.LastIndex(id, "-"); idx > 0 {
		return id[:idx]
	}
	return id
}

// deleteChunksByPrefix removes all indexed chunks whose path is under prefix.
// An empty prefix matches every path.
func (a *App) deleteChunksByPrefix(ctx context.Context, prefix string) error {
	var toDelete []string
	for _, path := range a.store.Paths() {
		if !isUnderPath(path, prefix) {
			continue
		}
		toDelete = append(toDelete, path)
		a.symbolIndex.RemoveByPath(path)
		a.callGraph.RemoveByPath(path)
		a.simhashIndex.RemoveByPath(path)
	}
	if err := a.store.DeleteByPaths(ctx, toDelete); err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}
	return nil
}

func isUnderPath(path, root string) bool {
	if root == "" {
		return path != ""
	}
	if !strings.HasPrefix(path, root) {
		return false
	}
	if len(path) == len(root) {
		return true
	}
	return path[len(root)] == filepath.Separator
}
