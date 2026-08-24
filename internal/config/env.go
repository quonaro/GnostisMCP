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
		},
		Web: Web{
			Port: envIntOr("GS_WEB_PORT", defaultWebPort),
		},
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.DataDir = filepath.Clean(cfg.DataDir)
	cfg.ProjectsDirPath = filepath.Clean(cfg.ProjectsDirPath)

	cfg.Memory = Memory{
		Cascade: loadProviderConfig("cascade"),
	}

	if err := validate(&cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	slog.Debug("config loaded from env", "data_dir", cfg.DataDir, "model", cfg.Embeddings.Model)
	return cfg, nil
}

func loadProviderConfig(name string) ProviderConfig {
	prefix := "GS_MEMORY_" + strings.ToUpper(name)
	cfg := ProviderConfig{
		Enabled:              envBoolOr(prefix+"_ENABLED", false),
		SourceDirs:           envListOr(prefix+"_SOURCE_DIRS", nil),
		MinUserMessageLength: envIntOr(prefix+"_MIN_MSG_LEN", 0),
	}

	if !cfg.Enabled {
		return cfg
	}

	if cfg.MinUserMessageLength == 0 {
		cfg.MinUserMessageLength = defaultMinUserMessageLength
	}
	if len(cfg.SourceDirs) == 0 {
		cfg.SourceDirs = existingDefaultSourceDirs(name)
	}

	return cfg
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

	if err := validateProvider("memory.cascade", cfg.Memory.Cascade); err != nil {
		return err
	}

	return nil
}

func validateProvider(prefix string, cfg ProviderConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.MinUserMessageLength < 0 {
		return fmt.Errorf("%s.min_user_message_length must be non-negative", prefix)
	}
	if len(cfg.SourceDirs) == 0 {
		return fmt.Errorf("%s.source_dirs is required when enabled", prefix)
	}
	for i, src := range cfg.SourceDirs {
		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("%s.source_dirs[%d] %s: %w", prefix, i, src, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s.source_dirs[%d] %s is not a directory", prefix, i, src)
		}
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

func envBoolOr(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

func envListOr(key string, defaultVal []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func existingDefaultSourceDirs(name string) []string {
	if name != "cascade" {
		return nil
	}
	var out []string
	for _, d := range DefaultMemorySourceDirs() {
		if _, err := os.Stat(d); err == nil {
			out = append(out, d)
		}
	}
	return out
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

// DefaultMemorySourceDirs returns the standard Windsurf/Next/Devin/Desktop
// Cascade trajectory directories if they exist on the current system.
func DefaultMemorySourceDirs() []string {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}
	base := filepath.Join(home, ".codeium")
	return []string{
		filepath.Join(base, "windsurf", "cascade"),
		filepath.Join(base, "windsurf-next", "cascade"),
		filepath.Join(base, "devin", "cascade"),
		filepath.Join(base, "devin-desktop", "cascade"),
	}
}
