package app

import (
	"context"
	"fmt"

	"github.com/quonaro/gnostis/internal/jobs"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/stats"
)

// Status returns the configured project names and current chunk count.
func (a *App) Status() ([]string, int) {
	snap := a.projectsSnapshot.Load()
	if snap == nil {
		return nil, a.store.Count()
	}
	names := make([]string, len(*snap))
	for i, p := range *snap {
		names[i] = p.Name
	}
	return names, a.store.Count()
}

// Info returns runtime metadata about the active provider and index.
func (a *App) Info() (provider, model string, symbols int) {
	model, _ = a.modelName.Load().(string)
	return a.provider.ModelName(), model, a.symbolIndex.Count()
}

// ProgressState returns the persisted rebuild progress state.
func (a *App) ProgressState() (progress.State, error) {
	if a.progress == nil {
		return progress.State{Status: progress.StatusIdle}, nil
	}
	return a.progress.Load()
}

// ProjectStats returns the current chunk count, last indexed time, and
// indexing configuration for each configured project.
func (a *App) ProjectStats(ctx context.Context) (map[string]stats.Project, error) {
	loaded, err := a.indexingStats.Load()
	if err != nil {
		return nil, fmt.Errorf("load stats: %w", err)
	}

	snap := a.projectsSnapshot.Load()
	if snap == nil {
		return map[string]stats.Project{}, nil
	}

	dirsSnap := a.dirsSnapshot.Load()

	out := make(map[string]stats.Project, len(*snap))
	for _, p := range *snap {
		count, err := a.store.CountByProject(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("count project %q: %w", p.Name, err)
		}
		stat := stats.Project{Path: p.Path, Chunks: count}
		if s, ok := loaded[p.Name]; ok {
			stat.LastIndexedAt = s.LastIndexedAt
			stat.Model = s.Model
		}
		if dirsSnap != nil {
			for _, d := range *dirsSnap {
				if d.Name == p.Name {
					stat.Extensions = d.Extensions
					stat.Include = d.Include
					stat.Exclude = d.Exclude
					stat.MaxFileSizeMB = d.MaxFileSizeMB
					break
				}
			}
		}
		out[p.Name] = stat
	}
	return out, nil
}

// FailProgress marks the current rebuild as failed.
func (a *App) FailProgress(err error) {
	if a.progress != nil {
		_ = a.progress.Fail(err)
	}
}

// Jobs returns a snapshot of all jobs in the queue (pending, running, recently completed).
func (a *App) Jobs() []jobs.Job {
	if a.jobQueue == nil {
		return nil
	}
	return a.jobQueue.Snapshot()
}

// ProjectPath returns the filesystem path for a project by name.
func (a *App) ProjectPath(name string) (string, error) {
	snap := a.projectsSnapshot.Load()
	if snap == nil {
		return "", fmt.Errorf("no projects loaded")
	}
	for _, p := range *snap {
		if p.Name == name {
			return p.Path, nil
		}
	}
	return "", fmt.Errorf("project %q not found", name)
}
