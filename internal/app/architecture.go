package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/graph"
)

// Architecture aggregates index, graph, and change data for a project.
func (a *App) Architecture(ctx context.Context, projectName string) (*graph.Architecture, error) {
	projectPath, err := a.ProjectPath(projectName)
	if err != nil {
		return nil, err
	}

	nodes := a.callGraph.NodesByProject(projectPath)
	_, edgeCount := a.callGraph.Count()

	arch := &graph.Architecture{
		Project:         projectName,
		Languages:       make(map[string]int),
		SymbolsByKind:   make(map[string]int),
		EntryPoints:     []graph.EntryPoint{},
		Packages:        []graph.PackageInfo{},
		Hotspots:        []graph.Hotspot{},
		RecentlyChanged: []graph.RecentlyChanged{},
	}

	fileSet := make(map[string]bool)
	dirFiles := make(map[string]map[string]bool)

	for _, n := range nodes {
		fileSet[n.Path] = true

		if n.Kind != "" {
			arch.SymbolsByKind[n.Kind]++
		}

		if graph.IsEntryPoint(n) {
			arch.EntryPoints = append(arch.EntryPoints, graph.EntryPoint{
				Symbol: n.Symbol,
				Path:   n.Path,
				Kind:   n.Kind,
			})
		}

		rel, err := filepath.Rel(projectPath, n.Path)
		if err == nil {
			segments := strings.Split(rel, string(filepath.Separator))
			if len(segments) > 0 && segments[0] != "." {
				pkg := segments[0]
				if dirFiles[pkg] == nil {
					dirFiles[pkg] = make(map[string]bool)
				}
				dirFiles[pkg][n.Path] = true
			}
		}
	}

	// Count unique files per language
	langFiles := make(map[string]map[string]bool)
	for path := range fileSet {
		lang := chunker.DetectLanguage(path)
		if lang == "" {
			continue
		}
		if langFiles[lang] == nil {
			langFiles[lang] = make(map[string]bool)
		}
		langFiles[lang][path] = true
	}
	for lang, files := range langFiles {
		arch.Languages[lang] = len(files)
	}

	// Packages
	for dir, files := range dirFiles {
		arch.Packages = append(arch.Packages, graph.PackageInfo{
			Name:  dir,
			Files: len(files),
		})
	}
	sort.Slice(arch.Packages, func(i, j int) bool {
		return arch.Packages[i].Files > arch.Packages[j].Files
	})

	// Hotspots
	inDegree, outDegree := a.callGraph.DegreeMaps()

	type nodeDegree struct {
		node     graph.Node
		incoming int
		outgoing int
	}
	degrees := make([]nodeDegree, 0, len(nodes))
	for _, n := range nodes {
		nd := nodeDegree{
			node:     n,
			incoming: inDegree[n.ID],
			outgoing: outDegree[n.ID],
		}
		if nd.incoming+nd.outgoing > 0 {
			degrees = append(degrees, nd)
		}
	}
	sort.Slice(degrees, func(i, j int) bool {
		return degrees[i].incoming+degrees[i].outgoing > degrees[j].incoming+degrees[j].outgoing
	})
	limit := 10
	if len(degrees) < limit {
		limit = len(degrees)
	}
	for i := 0; i < limit; i++ {
		arch.Hotspots = append(arch.Hotspots, graph.Hotspot{
			Symbol:   degrees[i].node.Symbol,
			Path:     degrees[i].node.Path,
			Incoming: degrees[i].incoming,
			Outgoing: degrees[i].outgoing,
		})
	}

	arch.TotalFiles = len(fileSet)
	arch.TotalSymbols = len(nodes)
	arch.TotalEdges = edgeCount

	// Recently changed
	changes, err := a.DetectChangesCached(ctx, projectName)
	if err != nil {
		slog.WarnContext(ctx, "architecture: detect changes failed", "error", err)
	} else {
		for _, c := range changes {
			if c.Status == "modified" || c.Status == "new" {
				arch.RecentlyChanged = append(arch.RecentlyChanged, graph.RecentlyChanged{
					Path:   c.Path,
					Status: c.Status,
				})
			}
		}
	}

	return arch, nil
}
