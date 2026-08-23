package app

import (
	"context"
	"testing"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/project"
)

func buildTestGraph() *graph.Graph {
	g := graph.New()
	chunks := []chunker.Chunk{
		{Path: "/proj/main.go", Symbol: "main", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "handler"}}},
		{Path: "/proj/handler.go", Symbol: "handler", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "validate"}}},
		{Path: "/proj/validate.go", Symbol: "validate", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "db"}}},
		{Path: "/proj/db.go", Symbol: "db", Kind: "function", Language: "go"},
		{Path: "/proj/unused.go", Symbol: "unused", Kind: "function", Language: "go"},
		{Path: "/proj/main_test.go", Symbol: "TestX", Kind: "function", Language: "go"},
		{Path: "/proj/exported.go", Symbol: "DoThing", Kind: "function", Language: "go"},
	}
	for _, c := range chunks {
		g.AddChunk(c)
	}
	return g
}

func TestTracePath_Found(t *testing.T) {
	a := &App{callGraph: buildTestGraph()}

	result, err := a.TracePath(context.Background(), "main", "db", "", 10)
	if err != nil {
		t.Fatalf("TracePath: %v", err)
	}
	if !result.Found {
		t.Fatal("expected path to be found")
	}
	if len(result.Hops) != 4 {
		t.Fatalf("expected 4 hops, got %d: %+v", len(result.Hops), result.Hops)
	}
	expected := []string{"main", "handler", "validate", "db"}
	for i, h := range result.Hops {
		if h.Symbol != expected[i] {
			t.Errorf("hop %d: expected %q, got %q", i, expected[i], h.Symbol)
		}
	}
}

func TestTracePath_ReverseNotFound(t *testing.T) {
	a := &App{callGraph: buildTestGraph()}

	result, err := a.TracePath(context.Background(), "db", "main", "", 10)
	if err != nil {
		t.Fatalf("TracePath: %v", err)
	}
	if result.Found {
		t.Fatal("expected no path from db to main")
	}
}

func TestTracePath_UnknownSymbol(t *testing.T) {
	a := &App{callGraph: buildTestGraph()}

	_, err := a.TracePath(context.Background(), "nonexistent", "db", "", 10)
	if err == nil {
		t.Fatal("expected error for unknown symbol")
	}
}

func TestTracePath_DepthCap(t *testing.T) {
	a := &App{callGraph: buildTestGraph()}

	result, err := a.TracePath(context.Background(), "main", "db", "", 2)
	if err != nil {
		t.Fatalf("TracePath: %v", err)
	}
	if result.Found {
		t.Fatal("expected no path with maxDepth=2 for a 3-hop chain")
	}
}

func TestTracePath_SameSymbol(t *testing.T) {
	a := &App{callGraph: buildTestGraph()}

	result, err := a.TracePath(context.Background(), "main", "main", "", 10)
	if err != nil {
		t.Fatalf("TracePath: %v", err)
	}
	if !result.Found {
		t.Fatal("expected found for same symbol")
	}
	if len(result.Hops) != 1 {
		t.Fatalf("expected 1 hop, got %d", len(result.Hops))
	}
}

func testAppWithGraph() *App {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	snap := &projs
	a := &App{callGraph: buildTestGraph()}
	a.projectsSnapshot.Store(snap)
	return a
}

func TestDeadCode_FindsUnused(t *testing.T) {
	a := testAppWithGraph()

	candidates, err := a.DeadCode(context.Background(), "test", "", 50)
	if err != nil {
		t.Fatalf("DeadCode: %v", err)
	}

	found := false
	for _, c := range candidates {
		if c.Symbol == "unused" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'unused' in dead code candidates")
	}
}

func TestDeadCode_ExcludesMain(t *testing.T) {
	a := testAppWithGraph()

	candidates, err := a.DeadCode(context.Background(), "test", "", 50)
	if err != nil {
		t.Fatalf("DeadCode: %v", err)
	}

	for _, c := range candidates {
		if c.Symbol == "main" {
			t.Error("main should be excluded as entry point")
		}
	}
}

func TestDeadCode_ExcludesTestSymbols(t *testing.T) {
	a := testAppWithGraph()

	candidates, err := a.DeadCode(context.Background(), "test", "", 50)
	if err != nil {
		t.Fatalf("DeadCode: %v", err)
	}

	for _, c := range candidates {
		if c.Symbol == "TestX" {
			t.Error("TestX should be excluded as entry point (test symbol)")
		}
	}
}

func TestDeadCode_ExcludesExported(t *testing.T) {
	a := testAppWithGraph()

	candidates, err := a.DeadCode(context.Background(), "test", "", 50)
	if err != nil {
		t.Fatalf("DeadCode: %v", err)
	}

	for _, c := range candidates {
		if c.Symbol == "DoThing" {
			t.Error("DoThing should be excluded as exported symbol")
		}
	}
}

func TestIsEntryPoint(t *testing.T) {
	tests := []struct {
		name string
		node graph.Node
		want bool
	}{
		{"main", graph.Node{Symbol: "main", Language: "go"}, true},
		{"init", graph.Node{Symbol: "init", Language: "go"}, true},
		{"TestX in test file", graph.Node{Symbol: "TestX", Path: "/foo_test.go", Language: "go"}, true},
		{"BenchmarkX in test file", graph.Node{Symbol: "BenchmarkX", Path: "/foo_test.go", Language: "go"}, true},
		{"exported Go", graph.Node{Symbol: "DoThing", Language: "go"}, true},
		{"unexported Go", graph.Node{Symbol: "doThing", Language: "go"}, false},
		{"unused", graph.Node{Symbol: "unused", Language: "go"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := graph.IsEntryPoint(tt.node); got != tt.want {
				t.Errorf("IsEntryPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
