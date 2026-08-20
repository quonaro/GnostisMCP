package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/quonaro/gnostis/internal/app"
	"github.com/quonaro/gnostis/internal/lock"
	"github.com/quonaro/gnostis/internal/log"
	mcp "github.com/quonaro/gnostis/internal/mcp"
)

// version is set by the build linker to the short git commit hash.
var version string

// logOutput is the current shared log destination; it is set to both stderr and
// ~/.gnostis/gnostis.log when the binary starts.
var logOutput io.Writer = os.Stderr

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level: %s", level)
	}
}

func setupLogOutput() io.Writer {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Stderr
	}
	dir := filepath.Join(home, ".gnostis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return os.Stderr
	}
	path := filepath.Join(dir, "gnostis.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return os.Stderr
	}
	return io.MultiWriter(os.Stderr, f)
}

func main() {
	logOutput = setupLogOutput()
	slog.SetDefault(slog.New(log.NewHandler(logOutput, slog.LevelInfo)))

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	// Try to acquire an exclusive lock on the data directory. If another
	// gnostis process already holds the lock, this process becomes a
	// stdio proxy that forwards all MCP messages to the primary instance
	// over HTTP. This allows multiple editor windows to share a single
	// gnostis process while keeping the stdio MCP config format.
	flock := lock.New(cfg.DataDir)
	if err := flock.TryLock(); err != nil {
		slog.Info("another gnostis instance is running, starting stdio proxy", "data_dir", cfg.DataDir)
		mcpURL := fmt.Sprintf("http://localhost:%d/mcp", cfg.Web.Port)
		if err := runStdioProxy(mcpURL); err != nil {
			fmt.Fprintf(os.Stderr, "stdio proxy: %v\n", err)
			os.Exit(1)
		}
		return
	}
	defer func() { _ = flock.Unlock() }()
	slog.Info("acquired data dir lock, running as primary", "data_dir", cfg.DataDir)

	mcp.SetVersion(version)
	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize app: %v\n", err)
		os.Exit(1)
	}

	if err := application.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
}
