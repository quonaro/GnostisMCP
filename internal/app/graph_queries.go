package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/quonaro/gnostis/internal/graph"
)

// TracePath BFS-searches the call graph from symbol `from` to symbol `to`.
func (a *App) TracePath(_ context.Context, from, to, projectName string, maxDepth int) (graph.TraceResult, error) {
	if from == "" || to == "" {
		return graph.TraceResult{}, fmt.Errorf("from and to are required")
	}

	if maxDepth <= 0 {
		maxDepth = 10
	}
	if maxDepth > 25 {
		maxDepth = 25
	}

	projectPath := ""
	if projectName != "" {
		pp, err := a.ProjectPath(projectName)
		if err != nil {
			return graph.TraceResult{}, fmt.Errorf("project %q: %w", projectName, err)
		}
		projectPath = pp
	}

	fromNodes := a.callGraph.ResolveSymbol(from, projectPath)
	if len(fromNodes) == 0 {
		return graph.TraceResult{}, fmt.Errorf("symbol %q not found in call graph", from)
	}

	toNodes := a.callGraph.ResolveSymbol(to, projectPath)
	if len(toNodes) == 0 {
		return graph.TraceResult{}, fmt.Errorf("symbol %q not found in call graph", to)
	}

	targetSet := make(map[string]bool, len(toNodes))
	for _, n := range toNodes {
		targetSet[n.ID] = true
	}

	if from == to {
		hops := []graph.TraceHop{nodeToHop(fromNodes[0])}
		return graph.TraceResult{Found: true, Hops: hops, Depth: 0}, nil
	}

	type queueItem struct {
		nodeID string
		path   []graph.Node
	}

	visited := make(map[string]bool)
	for _, n := range fromNodes {
		visited[n.ID] = true
	}

	queue := make([]queueItem, 0, len(fromNodes))
	for _, n := range fromNodes {
		queue = append(queue, queueItem{nodeID: n.ID, path: []graph.Node{n}})
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		depth := len(item.path)

		if depth > maxDepth {
			break
		}

		callees := a.callGraph.CalleesOf(item.nodeID)
		for _, calleeID := range callees {
			if targetSet[calleeID] {
				calleeNode, ok := a.callGraph.NodeByID(calleeID)
				if !ok {
					continue
				}
				fullPath := append(item.path, calleeNode)
				return graph.TraceResult{
					Found: true,
					Hops:  nodesToHops(fullPath),
					Depth: len(fullPath) - 1,
				}, nil
			}

			if !visited[calleeID] && depth < maxDepth {
				visited[calleeID] = true
				calleeNode, ok := a.callGraph.NodeByID(calleeID)
				if !ok {
					continue
				}
				newPath := make([]graph.Node, len(item.path)+1)
				copy(newPath, item.path)
				newPath[len(item.path)] = calleeNode
				queue = append(queue, queueItem{nodeID: calleeID, path: newPath})
			}
		}
	}

	return graph.TraceResult{Found: false}, nil
}

// DeadCode lists candidate dead symbols in a project.
func (a *App) DeadCode(_ context.Context, projectName, kind string, topK int) ([]graph.DeadCodeCandidate, error) {
	projectPath, err := a.ProjectPath(projectName)
	if err != nil {
		return nil, fmt.Errorf("project %q: %w", projectName, err)
	}

	if topK <= 0 {
		topK = 50
	}

	nodes := a.callGraph.NodesByProject(projectPath)

	kindFilter := map[string]bool{}
	if kind != "" {
		kindFilter[kind] = true
	} else {
		kindFilter["function"] = true
		kindFilter["method"] = true
	}

	inDegree, _ := a.callGraph.DegreeMaps()

	var candidates []graph.DeadCodeCandidate
	for _, n := range nodes {
		if !kindFilter[n.Kind] {
			continue
		}
		if graph.IsEntryPoint(n) {
			continue
		}
		if inDegree[n.ID] > 0 {
			continue
		}
		candidates = append(candidates, graph.DeadCodeCandidate{
			Symbol:    n.Symbol,
			Path:      n.Path,
			Kind:      n.Kind,
			StartLine: n.StartLine,
			EndLine:   n.EndLine,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Path < candidates[j].Path
	})

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	return candidates, nil
}

func nodeToHop(n graph.Node) graph.TraceHop {
	return graph.TraceHop{
		Symbol:    n.Symbol,
		Path:      n.Path,
		StartLine: n.StartLine,
		EndLine:   n.EndLine,
		Kind:      n.Kind,
	}
}

func nodesToHops(ns []graph.Node) []graph.TraceHop {
	hops := make([]graph.TraceHop, len(ns))
	for i, n := range ns {
		hops[i] = nodeToHop(n)
	}
	return hops
}

// GraphData returns raw nodes and edges for a project (or all projects if empty).
func (a *App) GraphData(projectName string) ([]graph.Node, []graph.Edge) {
	projectPath := ""
	if projectName != "" {
		pp, err := a.ProjectPath(projectName)
		if err != nil {
			return nil, nil
		}
		projectPath = pp
	}
	nodes := a.callGraph.NodesByProject(projectPath)
	edges := a.callGraph.EdgesByProject(projectPath)
	return nodes, edges
}
