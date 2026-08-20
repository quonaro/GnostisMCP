package main

import (
	"fmt"
	"log/slog"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/log"
)

func loadConfig() (config.Config, error) {
	cfg, err := config.FromEnv()
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

	return cfg, nil
}
