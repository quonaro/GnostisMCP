package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

// DB holds separate read and write connection pools for SQLite.
// WAL mode allows concurrent readers while a write is in progress,
// so reads are never blocked by indexing writes.
type DB struct {
	reader *sql.DB
	writer *sql.DB
}

// Writer returns the single-connection write pool.
func (d *DB) Writer() *sql.DB { return d.writer }

// Reader returns the multi-connection read pool.
func (d *DB) Reader() *sql.DB { return d.reader }

// Close closes both connection pools.
func (d *DB) Close() error {
	var firstErr error
	if err := d.reader.Close(); err != nil {
		firstErr = err
	}
	if err := d.writer.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// Open opens or creates a SQLite database at the given path and runs migrations.
// WAL mode is enabled for concurrent read access during writes.
// It returns a DB with separate read and write connection pools.
func Open(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"

	// Writer pool: single connection to serialise writes and avoid SQLITE_BUSY.
	writer, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite writer: %w", err)
	}
	writer.SetMaxOpenConns(1)

	if _, err := writer.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := writer.Exec(`PRAGMA wal_autocheckpoint=1000`); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("set wal autocheckpoint: %w", err)
	}
	if _, err := writer.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("set synchronous: %w", err)
	}

	if err := Migrate(writer); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	// Reader pool: multiple connections for concurrent reads in WAL mode.
	reader, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("open sqlite reader: %w", err)
	}
	maxReaders := runtime.NumCPU()
	if maxReaders < 4 {
		maxReaders = 4
	}
	reader.SetMaxOpenConns(maxReaders)

	slog.Info("sqlite database ready", "path", path, "readers", maxReaders)
	return &DB{reader: reader, writer: writer}, nil
}

// Migrate creates all required tables if they do not exist.
func Migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			name        TEXT PRIMARY KEY,
			path        TEXT NOT NULL,
			extensions  TEXT,
			include     TEXT,
			exclude     TEXT,
			max_file_mb INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS file_hashes (
			scope TEXT NOT NULL,
			path  TEXT NOT NULL,
			hash  TEXT NOT NULL,
			PRIMARY KEY (scope, path)
		)`,

		`CREATE TABLE IF NOT EXISTS embedding_dim (
			scope TEXT PRIMARY KEY,
			dim   INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS symbols (
			symbol     TEXT NOT NULL,
			path       TEXT NOT NULL,
			language   TEXT,
			kind       TEXT,
			start_line INTEGER,
			end_line   INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_symbol ON symbols(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_path ON symbols(path)`,

		`CREATE TABLE IF NOT EXISTS graph_nodes (
			id         TEXT PRIMARY KEY,
			path       TEXT NOT NULL,
			symbol     TEXT NOT NULL,
			kind       TEXT,
			language   TEXT,
			start_line INTEGER,
			end_line   INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_nodes_path ON graph_nodes(path)`,

		`CREATE TABLE IF NOT EXISTS graph_edges (
			from_id TEXT NOT NULL,
			"to"    TEXT NOT NULL,
			line    INTEGER,
			FOREIGN KEY (from_id) REFERENCES graph_nodes(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON graph_edges(from_id)`,

		`CREATE TABLE IF NOT EXISTS simhash_entries (
			fingerprint INTEGER NOT NULL,
			project_id  TEXT,
			path        TEXT NOT NULL,
			symbol      TEXT,
			start_line  INTEGER,
			end_line    INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_simhash_path ON simhash_entries(path)`,

		`CREATE TABLE IF NOT EXISTS progress_state (
			id            INTEGER PRIMARY KEY DEFAULT 1,
			job_id        TEXT,
			status        TEXT NOT NULL,
			phase         TEXT,
			project       TEXT,
			total_files   INTEGER,
			done_files    INTEGER,
			total_chunks  INTEGER,
			done_chunks   INTEGER,
			pid           INTEGER,
			started_at    TEXT,
			updated_at    TEXT,
			error         TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS project_stats (
			project         TEXT PRIMARY KEY,
			path            TEXT,
			chunks          INTEGER,
			last_indexed_at TEXT,
			model           TEXT
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec statement: %w\nstatement: %s", err, stmt)
		}
	}

	return nil
}
