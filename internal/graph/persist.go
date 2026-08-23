package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// persistData is the on-disk representation of the graph.
type persistData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Save writes the graph to a JSON file at the given path.
func (g *Graph) Save(path string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	data := persistData{
		Nodes: make([]Node, 0, len(g.nodes)),
		Edges: make([]Edge, len(g.edges)),
	}
	for _, n := range g.nodes {
		data.Nodes = append(data.Nodes, n)
	}
	copy(data.Edges, g.edges)

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}

// Load reads the graph from a JSON file at the given path.
func (g *Graph) Load(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var data persistData
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]Node, len(data.Nodes))
	g.files = make(map[string]bool, len(data.Nodes))
	for _, n := range data.Nodes {
		g.nodes[n.ID] = n
		g.files[n.Path] = true
	}
	g.edges = data.Edges

	return nil
}
