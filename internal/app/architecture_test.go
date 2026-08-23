package app

import (
	"context"
	"testing"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/project"
)

func buildArchGraph() *graph.Graph {
	g := graph.New()
	chunks := []chunker.Chunk{
		{Path: "/proj/internal/foo/a.go", Symbol: "main", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "handler"}, {Name: "validate"}, {Name: "db"}, {Name: "util"}, {Name: "log"}}},
		{Path: "/proj/internal/foo/b.go", Symbol: "handler", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "validate"}}},
		{Path: "/proj/internal/foo/c.go", Symbol: "validate", Kind: "function", Language: "go", Calls: []chunker.CallRef{{Name: "db"}}},
		{Path: "/proj/internal/foo/d.go", Symbol: "db", Kind: "function", Language: "go"},
		{Path: "/proj/scripts/run.py", Symbol: "run", Kind: "function", Language: "python"},
	}
	for _, c := range chunks {
		g.AddChunk(c)
	}
	return g
}

func TestArchitecture_Languages(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	if arch.Languages["go"] != 4 {
		t.Errorf("expected 4 go files, got %d", arch.Languages["go"])
	}
	if arch.Languages["python"] != 1 {
		t.Errorf("expected 1 python file, got %d", arch.Languages["python"])
	}
}

func TestArchitecture_Packages(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	if len(arch.Packages) < 2 {
		t.Fatalf("expected at least 2 packages, got %d", len(arch.Packages))
	}
	// internal should have more files than scripts
	if arch.Packages[0].Name != "internal" {
		t.Errorf("expected 'internal' first, got %q", arch.Packages[0].Name)
	}
}

func TestArchitecture_EntryPoints(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	found := false
	for _, ep := range arch.EntryPoints {
		if ep.Symbol == "main" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'main' in entry points")
	}
}

func TestArchitecture_Hotspots(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	if len(arch.Hotspots) == 0 {
		t.Fatal("expected at least one hotspot")
	}
	// main has 5 outgoing, should be first
	if arch.Hotspots[0].Symbol != "main" {
		t.Errorf("expected 'main' as top hotspot, got %q", arch.Hotspots[0].Symbol)
	}
}

func TestArchitecture_SymbolsByKind(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	if arch.SymbolsByKind["function"] != 5 {
		t.Errorf("expected 5 functions, got %d", arch.SymbolsByKind["function"])
	}
}

func TestArchitecture_Totals(t *testing.T) {
	projs := []project.Project{{Name: "test", Path: "/proj"}}
	a := &App{callGraph: buildArchGraph()}
	a.projectsSnapshot.Store(&projs)

	arch, err := a.Architecture(context.Background(), "test")
	if err != nil {
		t.Fatalf("Architecture: %v", err)
	}

	if arch.TotalFiles != 5 {
		t.Errorf("expected 5 files, got %d", arch.TotalFiles)
	}
	if arch.TotalSymbols != 5 {
		t.Errorf("expected 5 symbols, got %d", arch.TotalSymbols)
	}
	if arch.TotalEdges == 0 {
		t.Error("expected non-zero edges")
	}
}

func TestArchitecture_UnknownProject(t *testing.T) {
	a := &App{callGraph: graph.New()}

	_, err := a.Architecture(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
}
