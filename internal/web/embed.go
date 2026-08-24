package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

// handleSPA returns an http.Handler that serves embedded static files from
// dist/ with SPA fallback to index.html.
func (s *Server) handleSPA() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		slog.Error("create sub filesystem", "error", err)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "static files not available", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/" {
			data, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if _, err := fs.Stat(sub, cleanPath); err != nil {
			data, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
