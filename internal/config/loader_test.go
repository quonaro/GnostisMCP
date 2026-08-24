package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInterpolateEnv(t *testing.T) {
	t.Setenv("TEST_VAR", "world")

	cases := []struct {
		input    string
		expected string
	}{
		{"hello ${TEST_VAR}", "hello world"},
		{"${MISSING:-default}", "default"},
		{"${TEST_VAR:-fallback}", "world"},
		{"no vars", "no vars"},
	}

	for _, tc := range cases {
		got := InterpolateEnv(tc.input)
		if got != tc.expected {
			t.Errorf("interpolateEnv(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestInterpolateTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	if got := InterpolateEnv("~"); got != home {
		t.Errorf("InterpolateEnv(~) = %q, want %q", got, home)
	}
	want := filepath.Join(home, "foo", "bar")
	if got := InterpolateEnv("~/foo/bar"); got != want {
		t.Errorf("InterpolateEnv(~/foo/bar) = %q, want %q", got, want)
	}
}

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("GS_DATA_DIR", "/tmp/gnostis-test-data")
	t.Setenv("GS_PROJECTS_DIR", "/tmp/gnostis-test-projects")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}

	if cfg.Embeddings.URL != "http://localhost:7997/v1" {
		t.Errorf("default url = %q, want http://localhost:7997/v1", cfg.Embeddings.URL)
	}
	if cfg.Embeddings.BatchSize != 32 {
		t.Errorf("default batch size = %d, want 32", cfg.Embeddings.BatchSize)
	}
	if cfg.DataDir != "/tmp/gnostis-test-data" {
		t.Errorf("data_dir = %q, want /tmp/gnostis-test-data", cfg.DataDir)
	}
}
