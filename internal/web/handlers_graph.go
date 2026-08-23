package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/quonaro/gnostis/internal/coverage"
	"github.com/quonaro/gnostis/internal/graph"
)

type graphResponse = graph.LayoutResult

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")

	connectedOnly := true
	if v := r.URL.Query().Get("connected_only"); v == "false" || v == "0" {
		connectedOnly = false
	}

	maxNodes := 800
	if v := r.URL.Query().Get("max_nodes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxNodes = n
		}
	}

	result, err := s.app.GraphLayout(project, connectedOnly, maxNodes)
	if err != nil {
		slog.ErrorContext(r.Context(), "graph layout", "error", err, "project", project)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if result.Nodes == nil {
		result.Nodes = []graph.LayoutNode{}
	}
	if result.Edges == nil {
		result.Edges = []graph.ResolvedEdge{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleArchitecture(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}

	arch, err := s.app.Architecture(ctx, project)
	if err != nil {
		slog.ErrorContext(ctx, "architecture", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, arch)
}

func (s *Server) handleDeadCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}

	kind := r.URL.Query().Get("kind")
	topK := 50
	if v := r.URL.Query().Get("top_k"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topK = n
		}
	}

	candidates, err := s.app.DeadCode(ctx, project, kind, topK)
	if err != nil {
		slog.ErrorContext(ctx, "dead code", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if candidates == nil {
		candidates = []graph.DeadCodeCandidate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidates": candidates,
		"count":      len(candidates),
	})
}

func (s *Server) handleChanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}

	changes, err := s.app.DetectChanges(ctx, project)
	if err != nil {
		slog.ErrorContext(ctx, "detect changes", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if changes == nil {
		changes = []coverage.Change{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"changes": changes,
		"count":   len(changes),
	})
}
