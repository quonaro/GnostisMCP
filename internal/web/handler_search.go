package web

import (
	"log/slog"
	"net/http"
	"strconv"
)

type searchResult struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	Path      string  `json:"path"`
	Language  string  `json:"language"`
	Symbol    string  `json:"symbol"`
	Signature string  `json:"signature"`
	Content   string  `json:"content"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}

	topK := 10
	if v := r.URL.Query().Get("top_k"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topK = n
		}
	}

	projectFilter := r.URL.Query().Get("project")
	filters := map[string]string{}
	if projectFilter != "" {
		filters["project_id"] = projectFilter
	}

	results, err := s.search.Search(r.Context(), q, filters, topK)
	if err != nil {
		slog.ErrorContext(r.Context(), "search", "query", q, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]searchResult, 0, len(results))
	for _, res := range results {
		out = append(out, searchResult{
			ID:        res.ID,
			ProjectID: res.ProjectID,
			Path:      res.Path,
			Language:  res.Language,
			Symbol:    res.Symbol,
			Signature: res.Signature,
			Content:   res.Content,
			StartLine: res.StartLine,
			EndLine:   res.EndLine,
			Score:     res.Score,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": out,
		"count":   len(out),
	})
}
