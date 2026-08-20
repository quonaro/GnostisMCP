package web

import (
	"context"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

type pickDirectoryResponse struct {
	Path string `json:"path"`
}

func (s *Server) handlePickDirectory(w http.ResponseWriter, r *http.Request) {
	path, err := pickDirectory(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "pick directory", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if path == "" {
		writeJSON(w, http.StatusOK, pickDirectoryResponse{Path: ""})
		return
	}
	writeJSON(w, http.StatusOK, pickDirectoryResponse{Path: path})
}

func pickDirectory(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "linux":
		return pickDirectoryLinux(ctx)
	default:
		return "", nil
	}
}

func pickDirectoryLinux(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "zenity", "--file-selection", "--directory")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
