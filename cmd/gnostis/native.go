package main

import (
	"fmt"
	"log/slog"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/log"
)

func loadConfig() (config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}

	if cfg.LogLevel != "" {
		level, err := parseLogLevel(cfg.LogLevel)
		if err != nil {
			return config.Config{}, fmt.Errorf("parse log level: %w", err)
		}
		slog.SetDefault(slog.New(log.NewHandler(logOutput, level)))
	}

	if cfg.MCP.Version == "" {
		cfg.MCP.Version = version
	}

	return cfg, nil
}
