package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectsDirName is the subdirectory inside the config directory that holds
// individual project JSON files.
const ProjectsDirName = "projects"

// ProjectsDir returns the path to the projects directory next to the config
// file. The config file path must be resolved (absolute).
func ProjectsDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), ProjectsDirName)
}

// LoadProjectFiles reads all *.json files from the projects directory and
// returns them as Directory entries. A missing projects directory is not an
// error — it yields an empty slice.
func LoadProjectFiles(dir string) ([]Directory, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read projects dir %s: %w", dir, err)
	}

	var dirs []Directory
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		d, err := loadProjectFile(path)
		if err != nil {
			return nil, fmt.Errorf("load project file %s: %w", path, err)
		}
		dirs = append(dirs, d)
	}
	return dirs, nil
}

// loadProjectFile reads and parses a single project JSON file.
func loadProjectFile(path string) (Directory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Directory{}, fmt.Errorf("read file: %w", err)
	}

	var d Directory
	if err := json.Unmarshal(data, &d); err != nil {
		return Directory{}, fmt.Errorf("parse json: %w", err)
	}
	return d, nil
}

// SaveProjectFile writes a single project as a JSON file inside the projects
// directory. The directory is created if it does not exist.
func SaveProjectFile(dir string, d Directory) error {
	if d.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create projects dir: %w", err)
	}

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project json: %w", err)
	}

	path := projectFilePath(dir, d.Name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write project file %s: %w", path, err)
	}
	return nil
}

// DeleteProjectFile removes the JSON file for the given project name.
// A missing file is not an error.
func DeleteProjectFile(dir, name string) error {
	path := projectFilePath(dir, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete project file %s: %w", path, err)
	}
	return nil
}

func projectFilePath(dir, name string) string {
	return filepath.Join(dir, name+".json")
}
