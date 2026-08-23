package symbol

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

// Location describes a single symbol definition.
type Location struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	Symbol    string `json:"symbol"`
	Kind      string `json:"kind"`
	Signature string `json:"signature"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Chunk is a minimal subset of chunker.Chunk used to feed the symbol index.
type Chunk struct {
	ProjectID string
	Path      string
	Language  string
	Symbol    string
	Kind      string
	Signature string
	StartLine int
	EndLine   int
}

// Index maps symbol names to their definition locations.
// It is safe for concurrent use.
type Index struct {
	mu    sync.RWMutex
	data  map[string][]Location
	sqlDB *sql.DB
}

// New opens or creates a symbol index backed by SQLite.
func New(sqlDB *sql.DB) (*Index, error) {
	idx := &Index{
		data:  make(map[string][]Location),
		sqlDB: sqlDB,
	}
	if err := idx.load(); err != nil {
		return nil, fmt.Errorf("load symbol index: %w", err)
	}
	return idx, nil
}

// Add records a symbol location. Empty symbols are ignored.
func (idx *Index) Add(loc Location) {
	if loc.Symbol == "" {
		return
	}
	key := strings.ToLower(loc.Symbol)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.data[key] = append(idx.data[key], loc)
}

// AddChunks adds locations extracted from generic chunks that expose the
// required fields.
func (idx *Index) AddChunks(chunks []Chunk) {
	for _, ch := range chunks {
		if ch.Symbol == "" {
			continue
		}
		idx.Add(Location(ch))
	}
}

// RemoveByPath deletes all locations belonging to a file path.
func (idx *Index) RemoveByPath(path string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for key, locs := range idx.data {
		kept := locs[:0]
		for _, loc := range locs {
			if loc.Path != path {
				kept = append(kept, loc)
			}
		}
		if len(kept) == 0 {
			delete(idx.data, key)
		} else {
			idx.data[key] = kept
		}
	}
	if idx.sqlDB != nil {
		if _, err := idx.sqlDB.Exec(`DELETE FROM symbols WHERE path=?`, path); err != nil {
			fmt.Printf("WARN: delete symbols by path: %v\n", err)
		}
	}
}

// Lookup returns exact matches for a symbol name (case-insensitive).
func (idx *Index) Lookup(name string) []Location {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return append([]Location(nil), idx.data[strings.ToLower(name)]...)
}

// Count returns the total number of stored symbol locations.
func (idx *Index) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	total := 0
	for _, locs := range idx.data {
		total += len(locs)
	}
	return total
}

// SearchFuzzy returns locations whose symbol contains the query substring.
func (idx *Index) SearchFuzzy(query string) []Location {
	q := strings.ToLower(query)
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	var out []Location
	for key, locs := range idx.data {
		if key == q || strings.Contains(key, q) {
			out = append(out, locs...)
		}
	}
	return out
}

// Save persists the full index to SQLite. It replaces all rows in a single transaction.
func (idx *Index) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tx, err := idx.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM symbols`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete symbols: %w", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO symbols (symbol, path, language, kind, start_line, end_line) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for key, locs := range idx.data {
		for _, loc := range locs {
			if _, err := stmt.Exec(key, loc.Path, loc.Language, loc.Kind, loc.StartLine, loc.EndLine); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert symbol %s: %w", loc.Symbol, err)
			}
		}
	}
	return tx.Commit()
}

func (idx *Index) load() error {
	rows, err := idx.sqlDB.Query(`SELECT symbol, path, language, kind, start_line, end_line FROM symbols`)
	if err != nil {
		return fmt.Errorf("query symbols: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, path, language, kind string
		var startLine, endLine int
		if err := rows.Scan(&key, &path, &language, &kind, &startLine, &endLine); err != nil {
			return fmt.Errorf("scan symbol row: %w", err)
		}
		idx.data[key] = append(idx.data[key], Location{
			Symbol:    key,
			Path:      path,
			Language:  language,
			Kind:      kind,
			StartLine: startLine,
			EndLine:   endLine,
		})
	}
	return rows.Err()
}
