package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadProjectFile(t *testing.T) {
	dir := t.TempDir()

	d := Directory{
		Path:          "/some/path",
		Name:          "my-project",
		MaxFileSizeMB: 5,
	}

	if err := SaveProjectFile(dir, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(filepath.Join(dir, "my-project.json")); err != nil {
		t.Fatalf("project file not created: %v", err)
	}

	// Load and verify.
	dirs, err := LoadProjectFiles(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("expected 1 project, got %d", len(dirs))
	}
	if dirs[0].Name != "my-project" {
		t.Errorf("name = %q, want my-project", dirs[0].Name)
	}
	if dirs[0].Path != "/some/path" {
		t.Errorf("path = %q, want /some/path", dirs[0].Path)
	}
	if dirs[0].MaxFileSizeMB != 5 {
		t.Errorf("max_file_size_mb = %d, want 5", dirs[0].MaxFileSizeMB)
	}
}

func TestDeleteProjectFile(t *testing.T) {
	dir := t.TempDir()

	d := Directory{Path: "/x", Name: "to-delete"}
	if err := SaveProjectFile(dir, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := DeleteProjectFile(dir, "to-delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// File should be gone.
	if _, err := os.Stat(filepath.Join(dir, "to-delete.json")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, got err=%v", err)
	}

	// Deleting a non-existent file should not error.
	if err := DeleteProjectFile(dir, "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
}

func TestLoadProjectFilesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	dirs, err := LoadProjectFiles(dir)
	if err != nil {
		t.Fatalf("load from empty dir: %v", err)
	}
	if dirs != nil {
		t.Fatalf("expected nil, got %v", dirs)
	}
}

func TestLoadProjectFilesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	dirs, err := LoadProjectFiles(dir)
	if err != nil {
		t.Fatalf("load from missing dir: %v", err)
	}
	if dirs != nil {
		t.Fatalf("expected nil, got %v", dirs)
	}
}

func TestSaveProjectFileSanitizesName(t *testing.T) {
	dir := t.TempDir()

	d := Directory{Path: "/some/path", Name: "my/sub-project"}
	if err := SaveProjectFile(dir, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	// File should be flat, not in a subdirectory.
	if _, err := os.Stat(filepath.Join(dir, "my_sub-project.json")); err != nil {
		t.Fatalf("sanitized project file not created: %v", err)
	}

	// No subdirectory should exist.
	if _, err := os.Stat(filepath.Join(dir, "my")); !os.IsNotExist(err) {
		t.Fatalf("unexpected subdirectory created, err=%v", err)
	}

	// Delete should also use the sanitized name.
	if err := DeleteProjectFile(dir, "my/sub-project"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "my_sub-project.json")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, got err=%v", err)
	}
}

func TestLoadWithProjectFilesMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write config with inline directories.
	data := "directories:\n  - path: " + dir + "\n    name: test-proj\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Migration should have created a project JSON file.
	projFile := filepath.Join(ProjectsDir(path), "test-proj.json")
	if _, err := os.Stat(projFile); err != nil {
		t.Fatalf("migrated project file not created: %v", err)
	}

	if len(cfg.Directories) != 1 {
		t.Errorf("directories = %d, want 1", len(cfg.Directories))
	}
	if cfg.Directories[0].Name != "test-proj" {
		t.Errorf("name = %q, want test-proj", cfg.Directories[0].Name)
	}
}
