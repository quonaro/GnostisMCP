package graph

import (
	"testing"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/db"
)

func TestAddChunk(t *testing.T) {
	g := New()

	c := chunker.Chunk{
		Path:      "/foo/bar.go",
		Symbol:    "main",
		Kind:      "function",
		Language:  "go",
		StartLine: 1,
		EndLine:   10,
		Calls: []chunker.CallRef{
			{Name: "greet", Line: 3},
			{Name: "fmt.Println", Line: 4},
		},
	}

	g.AddChunk(c)

	nodes, edges := g.Count()
	if nodes != 1 {
		t.Errorf("expected 1 node, got %d", nodes)
	}
	if edges != 2 {
		t.Errorf("expected 2 edges, got %d", edges)
	}
}

func TestAddChunkSkipsNonCallable(t *testing.T) {
	g := New()

	g.AddChunk(chunker.Chunk{Path: "/foo.md", Symbol: "doc", Kind: "document"})
	g.AddChunk(chunker.Chunk{Path: "/foo.txt", Symbol: "file", Kind: "file"})
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "X", Kind: ""})

	nodes, _ := g.Count()
	if nodes != 0 {
		t.Errorf("expected 0 nodes for non-callable kinds, got %d", nodes)
	}
}

func TestRemoveByPath(t *testing.T) {
	g := New()

	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "A", Kind: "function", Calls: []chunker.CallRef{{Name: "B"}}})
	g.AddChunk(chunker.Chunk{Path: "/bar.go", Symbol: "B", Kind: "function"})

	nodes, edges := g.Count()
	if nodes != 2 || edges != 1 {
		t.Fatalf("expected 2 nodes, 1 edge; got %d nodes, %d edges", nodes, edges)
	}

	g.RemoveByPath("/foo.go")

	nodes, edges = g.Count()
	if nodes != 1 {
		t.Errorf("expected 1 node after remove, got %d", nodes)
	}
	if edges != 0 {
		t.Errorf("expected 0 edges after remove, got %d", edges)
	}
}

func TestResolveEdges(t *testing.T) {
	g := New()

	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "main", Kind: "function", Calls: []chunker.CallRef{{Name: "greet"}}})
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "greet", Kind: "function"})

	resolved := g.ResolveEdges()
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved edge, got %d", len(resolved))
	}
	if resolved[0].From != "/foo.go:main" {
		t.Errorf("expected from '/foo.go:main', got %q", resolved[0].From)
	}
	if resolved[0].To != "/foo.go:greet" {
		t.Errorf("expected to '/foo.go:greet', got %q", resolved[0].To)
	}
}

func TestCalleesAndCallers(t *testing.T) {
	g := New()

	g.AddChunk(chunker.Chunk{Path: "/a.go", Symbol: "main", Kind: "function", Calls: []chunker.CallRef{{Name: "helper"}}})
	g.AddChunk(chunker.Chunk{Path: "/a.go", Symbol: "helper", Kind: "function"})

	callees := g.CalleesOf("/a.go:main")
	if len(callees) != 1 || callees[0] != "/a.go:helper" {
		t.Errorf("unexpected callees: %v", callees)
	}

	callers := g.CallersOf("/a.go:helper")
	if len(callers) != 1 || callers[0] != "/a.go:main" {
		t.Errorf("unexpected callers: %v", callers)
	}
}

func TestSaveLoad(t *testing.T) {
	sqlDB := db.OpenTestDB(t)

	g := New()
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "main", Kind: "function", Calls: []chunker.CallRef{{Name: "greet"}}})
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "greet", Kind: "function"})

	if err := g.Save(sqlDB); err != nil {
		t.Fatalf("Save: %v", err)
	}

	g2 := New()
	if err := g2.Load(sqlDB); err != nil {
		t.Fatalf("Load: %v", err)
	}

	nodes, edges := g2.Count()
	if nodes != 2 {
		t.Errorf("expected 2 nodes after load, got %d", nodes)
	}
	if edges != 1 {
		t.Errorf("expected 1 edge after load, got %d", edges)
	}
}

func TestLoadMissingFile(t *testing.T) {
	sqlDB := db.OpenTestDB(t)
	g := New()
	if err := g.Load(sqlDB); err != nil {
		t.Errorf("Load on empty db should not error, got %v", err)
	}
}

func TestNodeID(t *testing.T) {
	id := NodeID("/foo.go", "main")
	if id != "/foo.go:main" {
		t.Errorf("expected '/foo.go:main', got %q", id)
	}
}

func TestSaveCreatesDir(t *testing.T) {
	sqlDB := db.OpenTestDB(t)

	g := New()
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "main", Kind: "function"})

	if err := g.Save(sqlDB); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestHasFile(t *testing.T) {
	g := New()

	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "A", Kind: "function"})
	g.AddChunk(chunker.Chunk{Path: "/bar.go", Symbol: "B", Kind: "function"})

	if !g.HasFile("/foo.go") {
		t.Error("expected HasFile /foo.go to be true")
	}
	if !g.HasFile("/bar.go") {
		t.Error("expected HasFile /bar.go to be true")
	}
	if g.HasFile("/missing.go") {
		t.Error("expected HasFile /missing.go to be false")
	}

	g.RemoveByPath("/foo.go")
	if g.HasFile("/foo.go") {
		t.Error("expected HasFile /foo.go to be false after remove")
	}
}

func TestHasFileAfterLoad(t *testing.T) {
	sqlDB := db.OpenTestDB(t)

	g := New()
	g.AddChunk(chunker.Chunk{Path: "/foo.go", Symbol: "main", Kind: "function"})
	g.AddChunk(chunker.Chunk{Path: "/bar.go", Symbol: "helper", Kind: "function"})

	if err := g.Save(sqlDB); err != nil {
		t.Fatalf("Save: %v", err)
	}

	g2 := New()
	if err := g2.Load(sqlDB); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !g2.HasFile("/foo.go") {
		t.Error("expected HasFile /foo.go to be true after Load")
	}
	if !g2.HasFile("/bar.go") {
		t.Error("expected HasFile /bar.go to be true after Load")
	}
	if g2.HasFile("/missing.go") {
		t.Error("expected HasFile /missing.go to be false after Load")
	}
}
