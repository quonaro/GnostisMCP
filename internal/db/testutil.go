package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// OpenTestDB opens an in-memory SQLite database with all migrations applied.
// It is intended for use in unit tests only.
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
