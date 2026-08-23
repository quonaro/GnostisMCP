package graph

import (
	"fmt"

	forcelib "github.com/ch-braun/go-spring-electrical-layout/force"
	liblayout "github.com/ch-braun/go-spring-electrical-layout/layout"
	"gonum.org/v1/gonum/graph"
	gnlayout "gonum.org/v1/gonum/graph/layout"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/spatial/r2"
)

// LayoutNode is a graph node with computed 2D coordinates and degree.
type LayoutNode struct {
	Node
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Degree int     `json:"degree"`
}

// LayoutResult is the complete graph layout response for the frontend.
type LayoutResult struct {
	Nodes         []LayoutNode   `json:"nodes"`
	Edges         []ResolvedEdge `json:"edges"`
	TotalNodes    int            `json:"total_nodes"`
	TotalEdges    int            `json:"total_edges"`
	Subsampled    bool           `json:"subsampled"`
	IsolatedCount int            `json:"isolated_count"`
}

// coordMap implements gonum's LayoutR2 interface for storing node positions.
type coordMap map[int64]r2.Vec

func (c coordMap) IsInitialized() bool            { return len(c) != 0 }
func (c coordMap) SetCoord2(id int64, pos r2.Vec) { c[id] = pos }
func (c coordMap) Coord2(id int64) r2.Vec         { return c[id] }

// ComputeLayout runs a spring-electrical force-directed layout on the given
// nodes and resolved edges, returning a map of node ID to [2]float64{x, y}.
// It uses a deterministic seed for reproducible results.
func ComputeLayout(nodes []Node, edges []ResolvedEdge) (map[string][2]float64, error) {
	if len(nodes) == 0 {
		return map[string][2]float64{}, nil
	}

	g := simple.NewWeightedUndirectedGraph(0, 0)

	// Map string node IDs to int64 gonum node IDs.
	idToInt := make(map[string]int64, len(nodes))
	for i, n := range nodes {
		gonumID := int64(i)
		g.AddNode(simple.Node(gonumID))
		idToInt[n.ID] = gonumID
	}

	// Add edges.
	for _, e := range edges {
		fromID, ok := idToInt[e.From]
		if !ok {
			continue
		}
		toID, ok := idToInt[e.To]
		if !ok {
			continue
		}
		if fromID == toID {
			continue
		}
		g.SetWeightedEdge(g.NewWeightedEdge(simple.Node(fromID), simple.Node(toID), 1.0))
	}

	// Configure force stack with parameters tuned for code graph visualization.
	const (
		seed          uint64  = 42
		updates       uint    = 300
		stepSize      float64 = 0.01
		coolingRate   float64 = 0.1
		stopThreshold float64 = 0.5
		optimalDist   float64 = 10.0
		repulsion     float64 = 10.0
		repulsionExp  uint    = 3
		attractionExp float64 = 1.0
		epsilon       float64 = 1e-4
		gravityConst  float64 = 100.0
	)

	stack := forcelib.NewForceStack(seed, updates, stepSize, coolingRate, stopThreshold)
	stack.AddForce(forcelib.NewSpringElectricalR2(optimalDist, repulsion, repulsionExp, attractionExp, epsilon))
	stack.AddForce(forcelib.NewCentralGravityR2(r2.Vec{X: 0, Y: 0}, gravityConst))

	// Use our own LayoutR2 implementation for deterministic position storage.
	coords := make(coordMap)
	for {
		cont := stack.Update(graph.Graph(g), gnlayout.LayoutR2(coords))
		if !cont {
			break
		}
	}

	// Extract positions.
	positions := make(map[string][2]float64, len(nodes))
	for _, n := range nodes {
		gonumID := idToInt[n.ID]
		coord, ok := coords[gonumID]
		if !ok {
			// Node had no position assigned (shouldn't happen).
			liblayout.AssignRandomCoordinates(g, coords, seed)
			coord = coords[gonumID]
		}
		positions[n.ID] = [2]float64{coord.X, coord.Y}
	}

	if len(positions) == 0 {
		return nil, fmt.Errorf("layout produced no positions")
	}

	return positions, nil
}
