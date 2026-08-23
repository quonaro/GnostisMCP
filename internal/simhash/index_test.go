package simhash

import (
	"path/filepath"
	"testing"
)

func TestIndex_AddAndFindSimilar(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewIndex(filepath.Join(dir, "simhash.json"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	text := "func process(data []byte) error { return nil }"
	fp := Fingerprint(text)

	idx.Add(fp, Meta{ProjectID: "p1", Path: "/a.go", Symbol: "process", StartLine: 1, EndLine: 3})

	matches := idx.FindSimilar(fp, 0.85, "/other.go", 5)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Meta.Path != "/a.go" {
		t.Errorf("expected match at /a.go, got %s", matches[0].Meta.Path)
	}
	if matches[0].Similarity != 1.0 {
		t.Errorf("expected similarity 1.0, got %f", matches[0].Similarity)
	}
}

func TestIndex_ExcludePath(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewIndex(filepath.Join(dir, "simhash.json"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	fp := Fingerprint("func helper() { return 42 }")
	idx.Add(fp, Meta{Path: "/a.go", Symbol: "helper"})

	matches := idx.FindSimilar(fp, 0.85, "/a.go", 5)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches when excluding /a.go, got %d", len(matches))
	}
}

func TestIndex_RemoveByPath(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewIndex(filepath.Join(dir, "simhash.json"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	fp := Fingerprint("func test() {}")
	idx.Add(fp, Meta{Path: "/a.go", Symbol: "test"})
	idx.Add(fp, Meta{Path: "/b.go", Symbol: "test"})

	idx.RemoveByPath("/a.go")

	matches := idx.FindSimilar(fp, 0.85, "/c.go", 10)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after remove, got %d", len(matches))
	}
	if matches[0].Meta.Path != "/b.go" {
		t.Errorf("expected /b.go, got %s", matches[0].Meta.Path)
	}
}

func TestIndex_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "simhash.json")

	idx, err := NewIndex(path)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	fp := Fingerprint("func saved() { return true }")
	idx.Add(fp, Meta{ProjectID: "p1", Path: "/x.go", Symbol: "saved", StartLine: 1, EndLine: 3})

	if err := idx.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	idx2, err := NewIndex(path)
	if err != nil {
		t.Fatalf("NewIndex reload: %v", err)
	}

	matches := idx2.FindSimilar(fp, 0.85, "/other.go", 5)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after reload, got %d", len(matches))
	}
	if matches[0].Meta.Symbol != "saved" {
		t.Errorf("expected symbol 'saved', got %s", matches[0].Meta.Symbol)
	}
}

func TestIndex_LoadMissingFile(t *testing.T) {
	idx, err := NewIndex("/nonexistent/path/simhash.json")
	if err != nil {
		t.Fatalf("NewIndex with missing file should not error: %v", err)
	}
	matches := idx.FindSimilar(0, 0.85, "", 5)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches from empty index, got %d", len(matches))
	}
}

func TestIndex_ThresholdFiltering(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewIndex(filepath.Join(dir, "simhash.json"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	similar := Fingerprint("func process(data []byte) error { return nil }")
	different := Fingerprint("class DatabaseConnection: def __init__(self):")

	idx.Add(similar, Meta{Path: "/a.go", Symbol: "process"})
	idx.Add(different, Meta{Path: "/b.go", Symbol: "DatabaseConnection"})

	query := Fingerprint("func process(data []byte) error { return nil }")

	matches := idx.FindSimilar(query, 0.99, "/q.go", 5)
	if len(matches) != 1 {
		t.Errorf("threshold 0.99 should match only identical, got %d", len(matches))
	}
	if matches[0].Meta.Path != "/a.go" {
		t.Errorf("expected /a.go, got %s", matches[0].Meta.Path)
	}
}
