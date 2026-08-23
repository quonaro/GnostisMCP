package graph

import (
	"strings"
	"unicode"
)

// ResolveSymbol returns nodes whose symbol matches the given name.
// If projectPath is non-empty, only nodes under that path prefix are returned.
func (g *Graph) ResolveSymbol(symbol, projectPath string) []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var out []Node
	for _, n := range g.nodes {
		if n.Symbol != symbol {
			continue
		}
		if projectPath != "" && !strings.HasPrefix(n.Path, projectPath) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// InDegree returns the number of resolved edges pointing to the given node ID.
func (g *Graph) InDegree(nodeID string) int {
	resolved := g.ResolveEdges()
	count := 0
	for _, e := range resolved {
		if e.To == nodeID {
			count++
		}
	}
	return count
}

// OutDegree returns the number of resolved edges from the given node ID.
func (g *Graph) OutDegree(nodeID string) int {
	resolved := g.ResolveEdges()
	count := 0
	for _, e := range resolved {
		if e.From == nodeID {
			count++
		}
	}
	return count
}

// IsEntryPoint returns true for symbols expected to have no callers.
func IsEntryPoint(n Node) bool {
	if n.Symbol == "main" || n.Symbol == "init" {
		return true
	}

	if strings.HasSuffix(n.Path, "_test.go") {
		if strings.HasPrefix(n.Symbol, "Test") ||
			strings.HasPrefix(n.Symbol, "Benchmark") ||
			strings.HasPrefix(n.Symbol, "Example") ||
			strings.HasPrefix(n.Symbol, "Fuzz") {
			return true
		}
	}

	if n.Language == "go" {
		for _, r := range n.Symbol {
			return unicode.IsUpper(r)
		}
	}

	return false
}

// NodesByProject returns all nodes whose path is under the given project path prefix.
func (g *Graph) NodesByProject(projectPath string) []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var out []Node
	for _, n := range g.nodes {
		if projectPath == "" || strings.HasPrefix(n.Path, projectPath) {
			out = append(out, n)
		}
	}
	return out
}

// ResolveEdges returns edges where the callee name matches a known node symbol.
// This is a simple name-based resolution — it does not handle overloading or
// package-qualified names.
func (g *Graph) ResolveEdges() []ResolvedEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	bySymbol := make(map[string][]Node)
	for _, n := range g.nodes {
		bySymbol[n.Symbol] = append(bySymbol[n.Symbol], n)
	}

	var out []ResolvedEdge
	for _, e := range g.edges {
		targets, ok := bySymbol[e.To]
		if !ok {
			continue
		}
		for _, t := range targets {
			out = append(out, ResolvedEdge{
				From:     e.From,
				To:       t.ID,
				ToSymbol: t.Symbol,
				ToPath:   t.Path,
				Line:     e.Line,
			})
		}
	}
	return out
}

// ResolvedEdge is an edge with the callee resolved to a concrete node.
type ResolvedEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	ToSymbol string `json:"to_symbol"`
	ToPath   string `json:"to_path"`
	Line     int    `json:"line"`
}

// AdjacencyList returns a map from node ID to the list of callee node IDs.
func (g *Graph) AdjacencyList() map[string][]string {
	resolved := g.ResolveEdges()
	out := make(map[string][]string)
	for _, e := range resolved {
		out[e.From] = append(out[e.From], e.To)
	}
	return out
}

// ReverseAdjacencyList returns a map from callee node ID to caller node IDs.
func (g *Graph) ReverseAdjacencyList() map[string][]string {
	resolved := g.ResolveEdges()
	out := make(map[string][]string)
	for _, e := range resolved {
		out[e.To] = append(out[e.To], e.From)
	}
	return out
}

// CalleesOf returns the node IDs called by the given node.
func (g *Graph) CalleesOf(nodeID string) []string {
	adj := g.AdjacencyList()
	return adj[nodeID]
}

// CallersOf returns the node IDs that call the given node.
func (g *Graph) CallersOf(nodeID string) []string {
	rev := g.ReverseAdjacencyList()
	return rev[nodeID]
}

// DegreeMaps returns in-degree and out-degree maps for all nodes in a single
// pass over resolved edges. This is O(E) instead of O(N*E) when calling
// InDegree/OutDegree per node.
func (g *Graph) DegreeMaps() (inDegree, outDegree map[string]int) {
	resolved := g.ResolveEdges()
	inDegree = make(map[string]int, len(g.nodes))
	outDegree = make(map[string]int, len(g.nodes))
	for _, e := range resolved {
		outDegree[e.From]++
		inDegree[e.To]++
	}
	return inDegree, outDegree
}

// EdgesByProject returns all edges whose source node is under the given project path prefix.
// If projectPath is empty, all edges are returned.
func (g *Graph) EdgesByProject(projectPath string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		if projectPath == "" {
			out = append(out, e)
			continue
		}
		if n, ok := g.nodes[e.From]; ok && strings.HasPrefix(n.Path, projectPath) {
			out = append(out, e)
		}
	}
	return out
}
