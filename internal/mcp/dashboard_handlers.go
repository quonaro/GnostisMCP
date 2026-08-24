package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DashboardHandlers returns the map of dashboard-only JSON-RPC methods
// that are not exposed as MCP tools.
func (s *Server) DashboardHandlers() map[string]DashboardHandler {
	return map[string]DashboardHandler{
		"pick_directory": s.handlePickDirectory,
		"open_project":   s.handleOpenProject,
	}
}

func (s *Server) handlePickDirectory(ctx context.Context, _ json.RawMessage) (any, error) {
	path, err := pickDirectory(ctx)
	if err != nil {
		return nil, fmt.Errorf("pick directory: %w", err)
	}
	return map[string]string{"path": path}, nil
}

func (s *Server) handleOpenProject(_ context.Context, params json.RawMessage) (any, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if s.indexer == nil {
		return nil, fmt.Errorf("indexer is not configured")
	}

	path, err := s.indexer.ProjectPath(args.Name)
	if err != nil {
		return nil, err
	}

	if err := openInFileManager(path); err != nil {
		return nil, fmt.Errorf("open file manager: %w", err)
	}
	return map[string]string{"status": "opened"}, nil
}

func pickDirectory(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.CommandContext(ctx, "zenity", "--file-selection", "--directory")
		out, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return "", nil
			}
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", nil
	}
}

func openInFileManager(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
