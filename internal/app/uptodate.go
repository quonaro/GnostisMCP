package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/quonaro/gnostis/internal/directory"
)

// isIndexUpToDate reports whether all indexed files are unchanged and no new
// indexable files have appeared since the given timestamp. It performs a
// lightweight mtime-based scan without reading file contents.
func (a *App) isIndexUpToDate(ctx context.Context, since time.Time) bool {
	for _, path := range a.store.Paths() {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
	}

	for _, dir := range a.dirs {
		changed, err := scanDirForChanges(dir, since)
		if err != nil {
			slog.WarnContext(ctx, "scan directory for changes", "dir", dir.Path, "error", err)
			return false
		}
		if changed {
			return false
		}
	}
	return true
}

// scanDirForChanges walks a directory and reports whether any indexable file
// has an mtime newer than since.
func scanDirForChanges(dir directory.Directory, since time.Time) (bool, error) {
	var found bool
	err := filepath.WalkDir(dir.Path, func(absPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dir.Path, absPath)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !dir.ShouldIndex(relPath, info.Size()) {
			return nil
		}
		if info.ModTime().After(since) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}
