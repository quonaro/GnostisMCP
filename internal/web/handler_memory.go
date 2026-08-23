package web

import (
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/quonaro/gnostis/internal/memory"
)

func (s *Server) handleMemoryFiles(w http.ResponseWriter, r *http.Request) {
	files := s.app.MemoryFiles(r.Context())
	if files == nil {
		files = []memory.FileInfo{}
	}
	writeJSON(w, http.StatusOK, files)
}

type openMemoryFileRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleOpenMemoryFile(w http.ResponseWriter, r *http.Request) {
	var req openMemoryFileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	dataDir := s.app.MemoryDataDir()
	if dataDir == "" {
		writeError(w, http.StatusServiceUnavailable, "memory is not enabled")
		return
	}

	clean := filepath.Clean(req.Path)
	if !strings.HasPrefix(clean, dataDir) {
		writeError(w, http.StatusForbidden, "path is outside memory directory")
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", clean)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", clean)
	default:
		cmd = exec.Command("xdg-open", clean)
	}
	if err := cmd.Start(); err != nil {
		slog.ErrorContext(r.Context(), "open memory file", "path", clean, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open file")
		return
	}
	_ = cmd.Process.Release()

	writeJSON(w, http.StatusOK, map[string]string{"status": "opened"})
}
