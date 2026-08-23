package app

import (
	"context"
	"time"

	"github.com/quonaro/gnostis/internal/coverage"
)

const changesCacheTTL = 30 * time.Second

type changesCacheEntry struct {
	changes   []coverage.Change
	err       error
	fetchedAt time.Time
}

// DetectChangesCached returns DetectChanges results with a 30-second TTL cache
// per project to avoid repeated filesystem walks on every architecture request.
func (a *App) DetectChangesCached(ctx context.Context, projectName string) ([]coverage.Change, error) {
	a.changesCacheMu.Lock()
	if a.changesCache != nil {
		if entry, ok := a.changesCache[projectName]; ok && time.Since(entry.fetchedAt) < changesCacheTTL {
			a.changesCacheMu.Unlock()
			return entry.changes, entry.err
		}
	}
	a.changesCacheMu.Unlock()

	changes, err := a.DetectChanges(ctx, projectName)

	a.changesCacheMu.Lock()
	if a.changesCache == nil {
		a.changesCache = make(map[string]changesCacheEntry)
	}
	a.changesCache[projectName] = changesCacheEntry{
		changes:   changes,
		err:       err,
		fetchedAt: time.Now(),
	}
	a.changesCacheMu.Unlock()

	return changes, err
}

// InvalidateChangesCache removes the cached DetectChanges result for a project.
// Called after reindexing to ensure stale data is not served.
func (a *App) InvalidateChangesCache(projectName string) {
	a.changesCacheMu.Lock()
	if a.changesCache != nil {
		delete(a.changesCache, projectName)
	}
	a.changesCacheMu.Unlock()
}

// invalidateAllChangesCaches clears the changes cache for every project.
func (a *App) invalidateAllChangesCaches() {
	a.changesCacheMu.Lock()
	if a.changesCache != nil {
		clear(a.changesCache)
	}
	a.changesCacheMu.Unlock()
}
