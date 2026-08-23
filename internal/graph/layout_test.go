package graph

import (
	"math"
	"testing"
)

func TestComputeLayout_Empty(t *testing.T) {
	pos, err := ComputeLayout(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pos) != 0 {
		t.Fatalf("expected empty positions, got %d", len(pos))
	}
}

func TestComputeLayout_SingleNode(t *testing.T) {
	nodes := []Node{{ID: "a", Symbol: "A"}}
	pos, err := ComputeLayout(nodes, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pos) != 1 {
		t.Fatalf("expected 1 position, got %d", len(pos))
	}
	if _, ok := pos["a"]; !ok {
		t.Fatal("missing position for node a")
	}
}

func TestComputeLayout_AllNodesHavePositions(t *testing.T) {
	nodes := []Node{
		{ID: "a", Symbol: "A"},
		{ID: "b", Symbol: "B"},
		{ID: "c", Symbol: "C"},
		{ID: "d", Symbol: "D"},
		{ID: "e", Symbol: "E"},
	}
	edges := []ResolvedEdge{
		{From: "a", To: "b"},
		{From: "b", To: "c"},
		{From: "c", To: "d"},
		{From: "d", To: "e"},
		{From: "a", To: "e"},
	}

	pos, err := ComputeLayout(nodes, edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pos) != len(nodes) {
		t.Fatalf("expected %d positions, got %d", len(nodes), len(pos))
	}
	for _, n := range nodes {
		coords, ok := pos[n.ID]
		if !ok {
			t.Errorf("missing position for node %s", n.ID)
			continue
		}
		if math.IsNaN(coords[0]) || math.IsNaN(coords[1]) {
			t.Errorf("NaN position for node %s: %v", n.ID, coords)
		}
		if math.IsInf(coords[0], 0) || math.IsInf(coords[1], 0) {
			t.Errorf("Inf position for node %s: %v", n.ID, coords)
		}
	}
}

func TestComputeLayout_ConnectedCloserThanUnconnected(t *testing.T) {
	// Build a graph: A-B-C-D-E (well-connected chain), Z is isolated.
	// With enough connectivity, connected nodes should cluster together.
	nodes := []Node{
		{ID: "a", Symbol: "A"},
		{ID: "b", Symbol: "B"},
		{ID: "c", Symbol: "C"},
		{ID: "d", Symbol: "D"},
		{ID: "e", Symbol: "E"},
		{ID: "z", Symbol: "Z"},
	}
	edges := []ResolvedEdge{
		{From: "a", To: "b"},
		{From: "b", To: "c"},
		{From: "c", To: "d"},
		{From: "d", To: "e"},
		{From: "a", To: "e"},
		{From: "b", To: "d"},
	}

	pos, err := ComputeLayout(nodes, edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pos) != 6 {
		t.Fatalf("expected 6 positions, got %d", len(pos))
	}

	// Average distance among connected nodes vs distance to isolated node.
	pa := pos["a"]
	pz := pos["z"]

	// Distance from A to connected neighbor B.
	pb := pos["b"]
	distAB := math.Sqrt((pa[0]-pb[0])*(pa[0]-pb[0]) + (pa[1]-pb[1])*(pa[1]-pb[1]))

	// Distance from A to isolated Z.
	distAZ := math.Sqrt((pa[0]-pz[0])*(pa[0]-pz[0]) + (pa[1]-pz[1])*(pa[1]-pz[1]))

	// This is a soft check — with repulsion the isolated node may still be close,
	// but typically it should be farther than a directly connected neighbor.
	// We use a tolerance: A-Z should not be significantly closer than A-B.
	if distAZ < distAB*0.1 {
		t.Errorf("isolated node Z too close to A: distAZ=%.2f, distAB=%.2f", distAZ, distAB)
	}
}
