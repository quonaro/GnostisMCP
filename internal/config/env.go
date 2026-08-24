package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// FromEnv reads all configuration from environment variables and applies defaults.
func FromEnv() (Config, error) {
	cfg := Config{
		LogLevel:        envOr("GS_LOG_LEVEL", defaultLogLevel),
		DataDir:         envOr("GS_DATA_DIR", InterpolateEnv(defaultDataDir)),
		ProjectsDirPath: envOr("GS_PROJECTS_DIR", InterpolateEnv(DefaultProjectsDir)),
		Embeddings: Embeddings{
			URL:       envOr("GS_EMBEDDINGS_URL", defaultURL),
			Model:     envOr("GS_EMBEDDINGS_MODEL", defaultModel),
			APIKey:    os.Getenv("GS_EMBEDDINGS_API_KEY"),
			BatchSize: envIntOr("GS_EMBEDDINGS_BATCH_SIZE", defaultBatchSize),
			MaxChars:  envIntOr("GS_EMBEDDINGS_MAX_CHARS", defaultMaxChars),
		},
		Web: Web{
			Port: envIntOr("GS_WEB_PORT", defaultWebPort),
		},
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.DataDir = filepath.Clean(cfg.DataDir)
	cfg.ProjectsDirPath = filepath.Clean(cfg.ProjectsDirPath)

	if err := validate(&cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	slog.Debug("config loaded from env", "data_dir", cfg.DataDir, "model", cfg.Embeddings.Model)
	return cfg, nil
}

func validate(cfg *Config) error {
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported log_level: %s", cfg.LogLevel)
	}

	if cfg.Embeddings.Model == "" {
		return fmt.Errorf("embeddings model is required")
	}

	if cfg.Embeddings.BatchSize <= 0 {
		return fmt.Errorf("embeddings batch_size must be positive")
	}

	return nil
}

func envOr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOr(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// InterpolateEnv expands ${VAR} and ${VAR:-default} patterns in a string.
func InterpolateEnv(input string) string {
	input = expandTilde(input)
	return envPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := envPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		name := parts[1]
		value := os.Getenv(name)
		if value != "" {
			return value
		}

		if len(parts) == 3 && parts[2] != "" {
			return parts[2]
		}

		return ""
	})
}

func expandTilde(input string) string {
	if input == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return input
		}
		return home
	}
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return input
		}
		return filepath.Join(home, input[2:])
	}
	return input
}
