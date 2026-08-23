package stats

import (
	"testing"
	"time"

	"github.com/quonaro/gnostis/internal/db"
)

func TestStatsUpdateAndLoad(t *testing.T) {
	s := New(db.OpenTestDB(t))

	if err := s.Update("foo", 42, "nomic-embed-text"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	stat, ok := loaded["foo"]
	if !ok {
		t.Fatal("expected project foo in stats")
	}
	if stat.Chunks != 42 {
		t.Fatalf("expected chunks 42, got %d", stat.Chunks)
	}
	if stat.LastIndexedAt.IsZero() {
		t.Fatal("expected non-zero last indexed time")
	}
}

func TestStatsLoadMissing(t *testing.T) {
	s := New(db.OpenTestDB(t))

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty map, got %d", len(loaded))
	}
}

func TestStatsUpdateOverwrites(t *testing.T) {
	s := New(db.OpenTestDB(t))

	_ = s.Update("foo", 10, "nomic-embed-text")
	before, _ := s.Load()
	first := before["foo"].LastIndexedAt

	time.Sleep(10 * time.Millisecond)
	_ = s.Update("foo", 20, "text-embedding-3-small")
	after, _ := s.Load()
	second := after["foo"].LastIndexedAt

	if !second.After(first) {
		t.Fatal("expected last indexed time to advance")
	}
	if after["foo"].Chunks != 20 {
		t.Fatalf("expected chunks 20, got %d", after["foo"].Chunks)
	}
}
