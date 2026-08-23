package app

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/quonaro/gnostis/internal/coverage"
	"github.com/quonaro/gnostis/internal/directory"
	"github.com/quonaro/gnostis/internal/indexer"
	"github.com/quonaro/gnostis/internal/project"
)

// CheckCoverage checks whether each path is indexed and up to date.
func (a *App) CheckCoverage(_ context.Context, paths []string) []coverage.Status {
	hashes := a.store.FileHashes()
	out := make([]coverage.Status, 0, len(paths))

	for _, p := range paths {
		st := coverage.Status{Path: p}

		info, err := os.Stat(p)
		if err != nil {
			if _, ok := hashes[p]; ok {
				st.Status = "deleted"
				st.IndexHash = hashes[p]
			} else {
				st.Status = "not_indexed"
			}
			out = append(out, st)
			continue
		}

		if info.IsDir() {
			st.Status = "not_indexed"
			out = append(out, st)
			continue
		}

		content, err := os.ReadFile(p)
		if err != nil {
			st.Status = "not_indexed"
			out = append(out, st)
			continue
		}

		fileHash := indexer.HashContent(content)
		st.FileHash = fileHash

		idxHash, ok := hashes[p]
		if !ok {
			st.Status = "not_indexed"
		} else if idxHash != fileHash {
			st.Status = "stale"
			st.IndexHash = idxHash
		} else {
			st.Status = "indexed"
			st.IndexHash = idxHash
		}

		out = append(out, st)
	}

	return out
}

// DetectChanges lists files that are new, modified, or deleted since the last
// index for the given project.
func (a *App) DetectChanges(ctx context.Context, projectName string) ([]coverage.Change, error) {
	dir, proj, err := a.findProject(projectName)
	if err != nil {
		return nil, err
	}

	files, err := a.indexer.Index(ctx, dir, proj)
	if err != nil {
		return nil, fmt.Errorf("walk project %q: %w", projectName, err)
	}

	storedHashes := a.store.FileHashes()
	currentHashes := make(map[string]string, len(files))
	for _, f := range files {
		currentHashes[f.Path] = f.Hash
	}

	var changes []coverage.Change

	for path, currentHash := range currentHashes {
		storedHash, ok := storedHashes[path]
		if !ok {
			changes = append(changes, coverage.Change{Path: path, Status: "new"})
		} else if storedHash != currentHash {
			changes = append(changes, coverage.Change{Path: path, Status: "modified"})
		}
	}

	for path := range storedHashes {
		if !strings.HasPrefix(path, dir.Path) {
			continue
		}
		if _, ok := currentHashes[path]; !ok {
			changes = append(changes, coverage.Change{Path: path, Status: "deleted"})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return changes, nil
}

func (a *App) findProject(name string) (directory.Directory, project.Project, error) {
	snap := a.projectsSnapshot.Load()
	dirsSnap := a.dirsSnapshot.Load()
	if snap == nil || dirsSnap == nil {
		return directory.Directory{}, project.Project{}, fmt.Errorf("no projects loaded")
	}
	for i, p := range *snap {
		if p.Name == name {
			return (*dirsSnap)[i], p, nil
		}
	}
	return directory.Directory{}, project.Project{}, fmt.Errorf("project %q not found", name)
}
