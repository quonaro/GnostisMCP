package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/stats"
)

type statusResponse struct {
	Projects     []string                 `json:"projects"`
	TotalChunks  int                      `json:"total_chunks"`
	Provider     string                   `json:"provider"`
	Model        string                   `json:"model"`
	Symbols      int                      `json:"symbols"`
	Progress     progress.State           `json:"progress"`
	ETA          string                   `json:"eta,omitempty"`
	ETASeconds   int64                    `json:"eta_seconds,omitempty"`
	ProjectStats map[string]stats.Project `json:"project_stats"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projects, chunks := s.app.Status()
	provider, model, symbols := s.app.Info()

	pstate, err := s.app.ProgressState()
	if err != nil {
		slog.ErrorContext(ctx, "get progress state", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pst, err := s.app.ProjectStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get project stats", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	eta := pstate.ETA()
	resp := statusResponse{
		Projects:     projects,
		TotalChunks:  chunks,
		Provider:     provider,
		Model:        model,
		Symbols:      symbols,
		Progress:     pstate,
		ProjectStats: pst,
	}
	if eta > 0 {
		resp.ETA = eta.String()
		resp.ETASeconds = int64(eta.Seconds())
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var lastPayload []byte
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	send := func() error {
		projects, chunks := s.app.Status()
		provider, model, symbols := s.app.Info()

		pstate, err := s.app.ProgressState()
		if err != nil {
			return fmt.Errorf("progress state: %w", err)
		}

		pst, err := s.app.ProjectStats(ctx)
		if err != nil {
			return fmt.Errorf("project stats: %w", err)
		}

		eta := pstate.ETA()
		resp := statusResponse{
			Projects:     projects,
			TotalChunks:  chunks,
			Provider:     provider,
			Model:        model,
			Symbols:      symbols,
			Progress:     pstate,
			ProjectStats: pst,
		}
		if eta > 0 {
			resp.ETA = eta.String()
			resp.ETASeconds = int64(eta.Seconds())
		}

		payload, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("marshal status: %w", err)
		}
		if string(payload) == string(lastPayload) {
			return nil
		}
		lastPayload = payload
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := send(); err != nil {
		slog.ErrorContext(ctx, "sse initial send", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := send(); err != nil {
				slog.DebugContext(ctx, "sse send", "error", err)
				return
			}
		}
	}
}
