package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// ProjectsDirName is the subdirectory inside the config directory that holds
// individual project JSON files.
const ProjectsDirName = "projects"

// ProjectsDir returns the path to the projects directory next to the config
// file. The config file path must be resolved (absolute).
func ProjectsDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), ProjectsDirName)
}

// LoadProjectFiles reads all projects from the SQLite database and returns
// them as Directory entries. An empty database yields an empty slice.
func LoadProjectFiles(db *sql.DB) ([]Directory, error) {
	rows, err := db.Query(`SELECT name, path, extensions, include, exclude, max_file_mb FROM projects`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var dirs []Directory
	for rows.Next() {
		var d Directory
		var extensions, include, exclude sql.NullString
		if err := rows.Scan(&d.Name, &d.Path, &extensions, &include, &exclude, &d.MaxFileSizeMB); err != nil {
			return nil, fmt.Errorf("scan project row: %w", err)
		}
		if extensions.Valid && extensions.String != "" {
			_ = json.Unmarshal([]byte(extensions.String), &d.Extensions)
		}
		if include.Valid && include.String != "" {
			_ = json.Unmarshal([]byte(include.String), &d.Include)
		}
		if exclude.Valid && exclude.String != "" {
			_ = json.Unmarshal([]byte(exclude.String), &d.Exclude)
		}
		dirs = append(dirs, d)
	}
	return dirs, rows.Err()
}

// SaveProjectFile writes a single project to the SQLite database.
func SaveProjectFile(db *sql.DB, d Directory) error {
	if d.Name == "" {
		return fmt.Errorf("project name is required")
	}

	var extensions, include, exclude string
	if len(d.Extensions) > 0 {
		b, _ := json.Marshal(d.Extensions)
		extensions = string(b)
	}
	if len(d.Include) > 0 {
		b, _ := json.Marshal(d.Include)
		include = string(b)
	}
	if len(d.Exclude) > 0 {
		b, _ := json.Marshal(d.Exclude)
		exclude = string(b)
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO projects (name, path, extensions, include, exclude, max_file_mb) VALUES (?, ?, ?, ?, ?, ?)`,
		d.Name, d.Path, extensions, include, exclude, d.MaxFileSizeMB)
	if err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}
	return nil
}

// DeleteProjectFile removes the project with the given name from the database.
// A missing row is not an error.
func DeleteProjectFile(db *sql.DB, name string) error {
	_, err := db.Exec(`DELETE FROM projects WHERE name=?`, name)
	if err != nil {
		return fmt.Errorf("delete project %s: %w", name, err)
	}
	return nil
}
