package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// startStatusTicker starts a goroutine that periodically collects status
// and broadcasts it to all connected WebSocket clients via JSON-RPC
// notifications. It stops when ctx is cancelled.
func (a *App) startStatusTicker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastHash string

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hash, payload := a.collectStatusPayload(ctx)
			if hash == lastHash {
				continue
			}
			lastHash = hash

			data, err := json.Marshal(payload)
			if err != nil {
				slog.ErrorContext(ctx, "status ticker marshal", "error", err)
				continue
			}

			var params map[string]any
			if err := json.Unmarshal(data, &params); err != nil {
				slog.ErrorContext(ctx, "status ticker unmarshal", "error", err)
				continue
			}

			a.mcp.SendNotificationToAll("gnostis/status", params)
		}
	}
}

// collectStatusPayload gathers the same data as get_index_status and returns
// a hash string (for change detection) and the payload as a map.
func (a *App) collectStatusPayload(ctx context.Context) (string, map[string]any) {
	projects, chunks := a.Status()
	provider, model, symbols := a.Info()

	pstate, err := a.ProgressState()
	if err != nil {
		slog.ErrorContext(ctx, "status ticker progress", "error", err)
		return "", nil
	}

	pst, err := a.ProjectStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "status ticker project stats", "error", err)
		return "", nil
	}

	jobList := a.Jobs()

	payload := map[string]any{
		"projects":      projects,
		"total_chunks":  chunks,
		"provider":      provider,
		"model":         model,
		"symbols":       symbols,
		"progress":      pstate,
		"project_stats": pst,
		"jobs":          jobList,
	}

	eta := pstate.ETA()
	if eta > 0 {
		payload["eta"] = eta.String()
		payload["eta_seconds"] = int64(eta.Seconds())
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", payload
	}
	return string(data), payload
}
