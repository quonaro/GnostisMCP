package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/quonaro/gnostis/internal/watcher"
)

// newWatcher creates a fresh filesystem watcher with the current directory list.
func (a *App) newWatcher() *watcher.Watcher {
	return watcher.New(a.dirs, func(path string) {
		desc := fmt.Sprintf("Auto-index: %s", filepath.Base(path))
		a.jobQueue.Submit("watcher", desc, func(jobCtx context.Context) error {
			a.rebuildMu.Lock()
			defer a.rebuildMu.Unlock()

			if err := reindexFile(jobCtx, path, a.dirs, a.projects, a.store, a.symbolIndex, a.callGraph, a.simhashIndex, a.provider, a.embeddingCache, a.indexingStats); err != nil {
				return fmt.Errorf("reindex file: %w", err)
			}
			if err := a.symbolIndex.Save(); err != nil {
				return fmt.Errorf("save symbol index: %w", err)
			}
			a.saveCallGraph()
			a.saveSimhashIndex()
			return nil
		})
	})
}

// restartWatcher stops the current watcher and starts a new one with the current
// directory list. It must be called with rebuildMu held.
func (a *App) restartWatcher(ctx context.Context) error {
	if !a.watcherStarted {
		a.watcher = a.newWatcher()
		return nil
	}

	if a.watcher != nil {
		if err := a.watcher.Stop(); err != nil {
			slog.ErrorContext(ctx, "stop watcher", "error", err)
		}
	}

	a.watcher = a.newWatcher()
	if err := a.watcher.Start(); err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	return nil
}

// scheduleWatcherRestart signals the debounced watcher restarter that a restart
// is needed. Multiple rapid calls coalesce into a single restart.
func (a *App) scheduleWatcherRestart() {
	select {
	case a.watcherRestartCh <- struct{}{}:
	default:
	}
}

// runWatcherRestarter debounces watcher restart requests. It waits for a
// quiet period after the last signal, then restarts the watcher once with
// the current directory list. This coalesces bursts of project edits
// (e.g. 34 concurrent edit_project MCP calls) into a single restart.
func (a *App) runWatcherRestarter(ctx context.Context) {
	const debounceDelay = 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.watcherRestartCh:
			timer := time.NewTimer(debounceDelay)
			drain := true
			for drain {
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-a.watcherRestartCh:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(debounceDelay)
				case <-timer.C:
					drain = false
				}
			}

			a.rebuildMu.Lock()
			if err := a.restartWatcher(ctx); err != nil {
				slog.ErrorContext(ctx, "debounced watcher restart", "error", err)
			}
			a.rebuildMu.Unlock()
		}
	}
}
