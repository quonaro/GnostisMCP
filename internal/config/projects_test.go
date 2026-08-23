package config

import (
	"testing"

	"github.com/quonaro/gnostis/internal/db"
)

func TestSaveAndLoadProjectFile(t *testing.T) {
	sqlDB := db.OpenTestDB(t)

	d := Directory{
		Path:          "/some/path",
		Name:          "my-project",
		MaxFileSizeMB: 5,
	}

	if err := SaveProjectFile(sqlDB, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load and verify.
	dirs, err := LoadProjectFiles(sqlDB)
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
	sqlDB := db.OpenTestDB(t)

	d := Directory{Path: "/x", Name: "to-delete"}
	if err := SaveProjectFile(sqlDB, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := DeleteProjectFile(sqlDB, "to-delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Row should be gone.
	dirs, err := LoadProjectFiles(sqlDB)
	if err != nil {
		t.Fatalf("load after delete: %v", err)
	}
	if len(dirs) != 0 {
		t.Fatalf("expected 0 projects after delete, got %d", len(dirs))
	}

	// Deleting a non-existent project should not error.
	if err := DeleteProjectFile(sqlDB, "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
}

func TestLoadProjectFilesEmptyDir(t *testing.T) {
	sqlDB := db.OpenTestDB(t)
	dirs, err := LoadProjectFiles(sqlDB)
	if err != nil {
		t.Fatalf("load from empty db: %v", err)
	}
	if dirs != nil {
		t.Fatalf("expected nil, got %v", dirs)
	}
}

func TestLoadProjectFilesMissingDir(t *testing.T) {
	sqlDB := db.OpenTestDB(t)
	dirs, err := LoadProjectFiles(sqlDB)
	if err != nil {
		t.Fatalf("load from empty db: %v", err)
	}
	if dirs != nil {
		t.Fatalf("expected nil, got %v", dirs)
	}
}

func TestSaveProjectFileSanitizesName(t *testing.T) {
	sqlDB := db.OpenTestDB(t)

	d := Directory{Path: "/some/path", Name: "my/sub-project"}
	if err := SaveProjectFile(sqlDB, d); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load and verify the name is preserved as-is (SQLite handles naming).
	dirs, err := LoadProjectFiles(sqlDB)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("expected 1 project, got %d", len(dirs))
	}
	if dirs[0].Name != "my/sub-project" {
		t.Errorf("name = %q, want my/sub-project", dirs[0].Name)
	}

	// Delete should also work with the original name.
	if err := DeleteProjectFile(sqlDB, "my/sub-project"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	dirs, err = LoadProjectFiles(sqlDB)
	if err != nil {
		t.Fatalf("load after delete: %v", err)
	}
	if len(dirs) != 0 {
		t.Fatalf("expected 0 projects after delete, got %d", len(dirs))
	}
}
