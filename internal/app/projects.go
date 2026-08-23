package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/directory"
	"github.com/quonaro/gnostis/internal/project"
)

// resolveProjects loads project JSON files from the projects directory and
// creates directory/project pairs.
func resolveProjects(cfg config.Config) ([]directory.Directory, []project.Project, error) {
	dirs, err := config.LoadProjectFiles(cfg.ProjectsDirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load project files: %w", err)
	}

	var outDirs []directory.Directory
	var outProjects []project.Project
	for _, d := range dirs {
		p := project.New(d.Name, d.Path)
		outDirs = append(outDirs, directory.FromConfig(d))
		outProjects = append(outProjects, p)
	}

	return outDirs, outProjects, nil
}

// AddProject saves a project JSON file to the projects directory and updates
// the in-memory project list. It does not start indexing — call
// StartRebuildProject separately to index the project.
func (a *App) AddProject(ctx context.Context, path, name string, extensions, include, exclude []string, maxFileSizeMB int) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", absPath)
	}

	a.rebuildMu.Lock()
	defer a.rebuildMu.Unlock()

	for _, p := range a.projects {
		if p.Path == absPath {
			return "", fmt.Errorf("project with path %q already exists", absPath)
		}
	}

	if name == "" {
		name = filepath.Base(absPath)
	}
	name = uniqueProjectName(name, a.projects)

	d := config.Directory{
		Path:          absPath,
		Name:          name,
		Extensions:    extensions,
		Include:       include,
		Exclude:       exclude,
		MaxFileSizeMB: maxFileSizeMB,
	}

	if err := config.SaveProjectFile(a.cfg.ProjectsDirPath, d); err != nil {
		return "", fmt.Errorf("save project file: %w", err)
	}

	a.dirs = append(a.dirs, directory.FromConfig(d))
	a.projects = append(a.projects, project.New(name, absPath))
	a.updateSnapshots(a.cfg, a.projects)

	if a.mcp != nil {
		a.mcp.ReloadProjects(a.projects)
	}
	a.scheduleWatcherRestart()
	return name, nil
}

// EditProject updates a project's configuration and saves it to the JSON file.
func (a *App) EditProject(ctx context.Context, name string, extensions, include, exclude []string, maxFileSizeMB int) error {
	a.rebuildMu.Lock()
	defer a.rebuildMu.Unlock()

	idx := -1
	for i, p := range a.projects {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("project %q not found", name)
	}

	d := config.Directory{
		Path:          a.projects[idx].Path,
		Name:          name,
		Extensions:    extensions,
		Include:       include,
		Exclude:       exclude,
		MaxFileSizeMB: maxFileSizeMB,
	}

	if err := config.SaveProjectFile(a.cfg.ProjectsDirPath, d); err != nil {
		return fmt.Errorf("save project file: %w", err)
	}

	a.dirs[idx] = directory.FromConfig(d)
	a.updateSnapshots(a.cfg, a.projects)

	if a.mcp != nil {
		a.mcp.ReloadProjects(a.projects)
	}
	a.scheduleWatcherRestart()
	return nil
}

// RemoveProject removes a project by name and deletes its indexed chunks.
func (a *App) RemoveProject(ctx context.Context, name string) error {
	a.rebuildMu.Lock()
	defer a.rebuildMu.Unlock()

	idx := -1
	for i, p := range a.projects {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("project %q not found", name)
	}

	path := a.projects[idx].Path

	if err := a.deleteChunksByPrefix(ctx, path); err != nil {
		return fmt.Errorf("delete project chunks: %w", err)
	}

	if err := a.symbolIndex.Save(); err != nil {
		return fmt.Errorf("save symbol index: %w", err)
	}

	a.dirs = append(a.dirs[:idx], a.dirs[idx+1:]...)
	a.projects = append(a.projects[:idx], a.projects[idx+1:]...)
	a.updateSnapshots(a.cfg, a.projects)

	if err := config.DeleteProjectFile(a.cfg.ProjectsDirPath, name); err != nil {
		return fmt.Errorf("delete project file: %w", err)
	}

	if a.mcp != nil {
		a.mcp.ReloadProjects(a.projects)
	}
	a.scheduleWatcherRestart()
	return nil
}

func uniqueProjectName(name string, projects []project.Project) string {
	used := make(map[string]bool)
	for _, p := range projects {
		used[p.Name] = true
	}
	if !used[name] {
		return name
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s-%d", name, n)
		if !used[candidate] {
			return candidate
		}
	}
}
