package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/directory"
	"github.com/quonaro/gnostis/internal/project"
)

// rebuildDirectory removes existing chunks under dirPath and reindexes the directory.
func (a *App) rebuildDirectory(ctx context.Context, dirPath string) error {
	slog.InfoContext(ctx, "rebuilding directory", "path", dirPath)

	if err := a.deleteChunksByPrefix(ctx, dirPath); err != nil {
		return fmt.Errorf("delete directory chunks: %w", err)
	}

	dir := directory.FromConfig(config.Directory{Path: dirPath, Name: filepath.Base(dirPath)})
	proj := project.New(filepath.Base(dirPath), dirPath)

	if err := indexDirectoryWithRetry(ctx, a.ProgressWriter, dir, proj, a.indexer, a.chunker, a.provider, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.embeddingCache, nil, a.indexingStats); err != nil {
		return fmt.Errorf("index directory: %w", err)
	}

	slog.InfoContext(ctx, "directory rebuild complete", "path", dirPath, "chunks", a.store.Count())
	return nil
}

// rebuildFile removes existing chunks for a single file and reindexes it.
func (a *App) rebuildFile(ctx context.Context, filePath string) error {
	_ = a.store.DeleteByPath(ctx, filePath)
	a.symbolIndex.RemoveByPath(filePath)

	if err := reindexFile(ctx, filePath, a.dirs, a.projects, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.provider, a.embeddingCache, a.indexingStats); err != nil {
		return fmt.Errorf("reindex file: %w", err)
	}
	return nil
}

// ReindexFiles reindexes the given file or directory paths and persists the symbol index.
// Paths outside configured directories are indexed with global defaults.
func (a *App) ReindexFiles(ctx context.Context, paths []string) error {
	a.rebuildMu.Lock()
	defer a.rebuildMu.Unlock()

	for _, raw := range paths {
		path, err := filepath.Abs(raw)
		if err != nil {
			return fmt.Errorf("resolve path %q: %w", raw, err)
		}

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		if info.IsDir() {
			if err := a.rebuildDirectory(ctx, path); err != nil {
				return fmt.Errorf("reindex directory %s: %w", path, err)
			}
			continue
		}

		if err := a.rebuildFile(ctx, path); err != nil {
			return fmt.Errorf("reindex file %s: %w", path, err)
		}
	}
	if err := a.symbolIndex.Save(); err != nil {
		return fmt.Errorf("save symbol index: %w", err)
	}
	a.saveCallGraph()
	a.saveSimhashIndex()
	a.invalidateAllChangesCaches()
	a.invalidateAllLayoutCaches()
	return nil
}
