package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/quonaro/gnostis/internal/graph"
)

type layoutCacheEntry struct {
	result graph.LayoutResult
}

// defaultMaxNodes is the maximum number of nodes to render in the graph view.
const defaultMaxNodes = 800

// defaultMaxEdges is the maximum number of edges to render.
const defaultMaxEdges = 4000

// GraphLayout returns a force-directed layout for the given project.
// It resolves edges, filters isolated nodes (if connectedOnly), subsamples
// by degree if the graph is too large, and computes 2D positions.
// Results are cached in-memory and invalidated on reindex/rebuild.
func (a *App) GraphLayout(projectName string, connectedOnly bool, maxNodes int) (graph.LayoutResult, error) {
	if maxNodes <= 0 {
		maxNodes = defaultMaxNodes
	}

	cacheKey := fmt.Sprintf("%s|%t|%d", projectName, connectedOnly, maxNodes)

	a.layoutCacheMu.Lock()
	if a.layoutCache != nil {
		if entry, ok := a.layoutCache[cacheKey]; ok {
			a.layoutCacheMu.Unlock()
			return entry.result, nil
		}
	}
	a.layoutCacheMu.Unlock()

	// Get raw nodes and edges.
	nodes, _ := a.GraphData(projectName)
	totalNodes := len(nodes)

	// Resolve edges to node IDs.
	resolvedEdges := a.callGraph.ResolveEdges()
	if projectName != "" {
		pp, err := a.ProjectPath(projectName)
		if err != nil {
			return graph.LayoutResult{}, fmt.Errorf("project %q: %w", projectName, err)
		}
		// Filter edges to those whose source is in the project.
		resolvedEdges = filterEdgesByProject(resolvedEdges, nodes, pp)
	}

	// Build node ID set for filtering.
	nodeIDs := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeIDs[n.ID] = true
	}

	// Filter edges to only those connecting known nodes.
	var workingEdges []graph.ResolvedEdge
	for _, e := range resolvedEdges {
		if nodeIDs[e.From] && nodeIDs[e.To] {
			workingEdges = append(workingEdges, e)
		}
	}
	totalEdges := len(workingEdges)

	// Compute degree for each node.
	degree := make(map[string]int, len(nodes))
	for _, e := range workingEdges {
		degree[e.From]++
		degree[e.To]++
	}

	// Filter isolated nodes if enabled.
	isolatedCount := 0
	workingNodes := nodes
	if connectedOnly {
		var connected []graph.Node
		for _, n := range nodes {
			if degree[n.ID] > 0 {
				connected = append(connected, n)
			}
		}
		isolatedCount = len(nodes) - len(connected)
		workingNodes = connected

		// Recompute edges with only connected nodes.
		connectedIDs := make(map[string]bool, len(workingNodes))
		for _, n := range workingNodes {
			connectedIDs[n.ID] = true
		}
		workingEdges = filterEdgesByIDs(workingEdges, connectedIDs)
	}

	// Subsample by degree if too many nodes.
	subsampled := false
	if len(workingNodes) > maxNodes {
		subsampled = true
		sort.Slice(workingNodes, func(i, j int) bool {
			return degree[workingNodes[i].ID] > degree[workingNodes[j].ID]
		})
		workingNodes = workingNodes[:maxNodes]

		keptIDs := make(map[string]bool, len(workingNodes))
		for _, n := range workingNodes {
			keptIDs[n.ID] = true
		}
		workingEdges = filterEdgesByIDs(workingEdges, keptIDs)
	}

	// Cap edges — keep highest-degree connections.
	if len(workingEdges) > defaultMaxEdges {
		sort.Slice(workingEdges, func(i, j int) bool {
			return (degree[workingEdges[i].From] + degree[workingEdges[i].To]) >
				(degree[workingEdges[j].From] + degree[workingEdges[j].To])
		})
		workingEdges = workingEdges[:defaultMaxEdges]
	}

	// Final pass: recompute degree and drop nodes that lost all connections.
	if connectedOnly {
		finalDegree := make(map[string]int, len(workingNodes))
		for _, e := range workingEdges {
			finalDegree[e.From]++
			finalDegree[e.To]++
		}
		var stillConnected []graph.Node
		for _, n := range workingNodes {
			if finalDegree[n.ID] > 0 {
				stillConnected = append(stillConnected, n)
			}
		}
		if len(stillConnected) < len(workingNodes) {
			isolatedCount += len(workingNodes) - len(stillConnected)
			workingNodes = stillConnected

			connectedIDs := make(map[string]bool, len(workingNodes))
			for _, n := range workingNodes {
				connectedIDs[n.ID] = true
			}
			workingEdges = filterEdgesByIDs(workingEdges, connectedIDs)
		}
		degree = finalDegree
	}

	// Compute layout positions.
	positions, err := graph.ComputeLayout(workingNodes, workingEdges)
	if err != nil {
		return graph.LayoutResult{}, fmt.Errorf("compute layout: %w", err)
	}

	// Build layout nodes with positions and degrees.
	layoutNodes := make([]graph.LayoutNode, 0, len(workingNodes))
	for _, n := range workingNodes {
		pos := positions[n.ID]
		layoutNodes = append(layoutNodes, graph.LayoutNode{
			Node:   n,
			X:      pos[0],
			Y:      pos[1],
			Degree: degree[n.ID],
		})
	}

	result := graph.LayoutResult{
		Nodes:         layoutNodes,
		Edges:         workingEdges,
		TotalNodes:    totalNodes,
		TotalEdges:    totalEdges,
		Subsampled:    subsampled,
		IsolatedCount: isolatedCount,
	}

	// Cache the result.
	a.layoutCacheMu.Lock()
	if a.layoutCache == nil {
		a.layoutCache = make(map[string]layoutCacheEntry)
	}
	a.layoutCache[cacheKey] = layoutCacheEntry{result: result}
	a.layoutCacheMu.Unlock()

	return result, nil
}

// InvalidateLayoutCache removes cached layout results for a project.
func (a *App) InvalidateLayoutCache(projectName string) {
	a.layoutCacheMu.Lock()
	if a.layoutCache != nil {
		for k := range a.layoutCache {
			if strings.HasPrefix(k, projectName+"|") {
				delete(a.layoutCache, k)
			}
		}
	}
	a.layoutCacheMu.Unlock()
}

// invalidateAllLayoutCaches clears all cached layout results.
func (a *App) invalidateAllLayoutCaches() {
	a.layoutCacheMu.Lock()
	if a.layoutCache != nil {
		clear(a.layoutCache)
	}
	a.layoutCacheMu.Unlock()
}

func filterEdgesByProject(edges []graph.ResolvedEdge, nodes []graph.Node, projectPath string) []graph.ResolvedEdge {
	nodePaths := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodePaths[n.ID] = n.Path
	}
	var out []graph.ResolvedEdge
	for _, e := range edges {
		path, ok := nodePaths[e.From]
		if !ok {
			continue
		}
		if strings.HasPrefix(path, projectPath) {
			out = append(out, e)
		}
	}
	return out
}

func filterEdgesByIDs(edges []graph.ResolvedEdge, ids map[string]bool) []graph.ResolvedEdge {
	var out []graph.ResolvedEdge
	for _, e := range edges {
		if ids[e.From] && ids[e.To] {
			out = append(out, e)
		}
	}
	return out
}
