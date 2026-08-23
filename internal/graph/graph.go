package graph

import (
	"sync"

	"github.com/quonaro/gnostis/internal/chunker"
)

// Node represents a callable symbol in the call graph.
type Node struct {
	ID        string `json:"id"` // unique key: path:symbol
	Path      string `json:"path"`
	Symbol    string `json:"symbol"`
	Kind      string `json:"kind"`
	Language  string `json:"language"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Edge represents a call from one symbol to another.
type Edge struct {
	From string `json:"from"` // caller node ID
	To   string `json:"to"`   // callee name (resolved at query time)
	Line int    `json:"line"`
}

// Graph is an in-memory call graph.
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]Node
	edges []Edge
	files map[string]bool
}

// New creates an empty Graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]Node),
		files: make(map[string]bool),
	}
}

// NodeID builds a deterministic node key from path and symbol.
func NodeID(path, symbol string) string {
	return path + ":" + symbol
}

// TraceHop is one node on a call path.
type TraceHop struct {
	Symbol    string `json:"symbol"`
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Kind      string `json:"kind"`
}

// TraceResult is the outcome of a trace_path query.
type TraceResult struct {
	Found bool       `json:"found"`
	Hops  []TraceHop `json:"hops,omitempty"`
	Depth int        `json:"depth,omitempty"`
}

// DeadCodeCandidate is a symbol with no detected callers.
type DeadCodeCandidate struct {
	Symbol    string `json:"symbol"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Architecture is a one-call structural overview of a project.
type Architecture struct {
	Project         string            `json:"project"`
	TotalFiles      int               `json:"total_files"`
	TotalSymbols    int               `json:"total_symbols"`
	TotalEdges      int               `json:"total_edges"`
	Languages       map[string]int    `json:"languages"`
	Packages        []PackageInfo     `json:"packages"`
	EntryPoints     []EntryPoint      `json:"entry_points"`
	Hotspots        []Hotspot         `json:"hotspots"`
	SymbolsByKind   map[string]int    `json:"symbols_by_kind"`
	RecentlyChanged []RecentlyChanged `json:"recently_changed,omitempty"`
}

// PackageInfo describes a top-level package directory.
type PackageInfo struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
}

// EntryPoint is a symbol that serves as an entry point.
type EntryPoint struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
	Kind   string `json:"kind"`
}

// Hotspot is a highly connected symbol.
type Hotspot struct {
	Symbol   string `json:"symbol"`
	Path     string `json:"path"`
	Incoming int    `json:"incoming"`
	Outgoing int    `json:"outgoing"`
}

// RecentlyChanged is a file that changed since last index.
type RecentlyChanged struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// AddChunk registers a chunk as a node and its calls as edges.
func (g *Graph) AddChunk(c chunker.Chunk) {
	if c.Kind == "" || c.Kind == "file" || c.Kind == "document" {
		return
	}

	id := NodeID(c.Path, c.Symbol)
	g.mu.Lock()
	defer g.mu.Unlock()

	g.files[c.Path] = true
	g.nodes[id] = Node{
		ID:        id,
		Path:      c.Path,
		Symbol:    c.Symbol,
		Kind:      c.Kind,
		Language:  c.Language,
		StartLine: c.StartLine,
		EndLine:   c.EndLine,
	}

	for _, call := range c.Calls {
		g.edges = append(g.edges, Edge{
			From: id,
			To:   call.Name,
			Line: call.Line,
		})
	}
}

// RemoveByPath removes all nodes and edges for the given file path.
func (g *Graph) RemoveByPath(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	toRemove := make(map[string]bool)
	found := false
	for id, n := range g.nodes {
		if n.Path == path {
			delete(g.nodes, id)
			toRemove[id] = true
			found = true
		}
	}
	if found {
		delete(g.files, path)
	}

	filtered := g.edges[:0]
	for _, e := range g.edges {
		if toRemove[e.From] {
			continue
		}
		filtered = append(filtered, e)
	}
	g.edges = filtered
}

// Nodes returns a copy of all nodes.
func (g *Graph) Nodes() []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	return out
}

// Edges returns a copy of all edges.
func (g *Graph) Edges() []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Edge, len(g.edges))
	copy(out, g.edges)
	return out
}

// Count returns the number of nodes and edges.
func (g *Graph) Count() (nodes, edges int) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes), len(g.edges)
}

// NodeByID returns the node with the given ID and whether it exists.
func (g *Graph) NodeByID(id string) (Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	return n, ok
}

// HasFile returns true if the graph has at least one node for the given file path.
func (g *Graph) HasFile(path string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.files[path]
}
