package simhash

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
)

// Meta identifies one chunk in the simhash index.
type Meta struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
	Symbol    string `json:"symbol"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Match is a similar chunk pair.
type Match struct {
	Meta       Meta    `json:"meta"`
	Similarity float64 `json:"similarity"`
}

// FileMatch groups matches per chunk of the queried file.
type FileMatch struct {
	Symbol    string  `json:"symbol"`
	StartLine int     `json:"start_line"`
	Matches   []Match `json:"matches"`
}

type indexEntry struct {
	Fingerprint uint64
	Meta        Meta
}

// Index maps simhash fingerprints to chunk metadata.
// It is safe for concurrent use.
// Scan is O(n) per query; LSH banding is a future optimization.
type Index struct {
	mu      sync.RWMutex
	entries []indexEntry
	sqlDB   *sql.DB
}

// NewIndex opens or creates a simhash index backed by SQLite.
func NewIndex(sqlDB *sql.DB) (*Index, error) {
	idx := &Index{sqlDB: sqlDB}
	if err := idx.load(); err != nil {
		return nil, fmt.Errorf("load simhash index: %w", err)
	}
	return idx, nil
}

// Add records a fingerprint with its chunk metadata.
func (idx *Index) Add(fp uint64, m Meta) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = append(idx.entries, indexEntry{Fingerprint: fp, Meta: m})
}

// RemoveByPath deletes all entries belonging to a file path.
func (idx *Index) RemoveByPath(path string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	kept := idx.entries[:0]
	for _, e := range idx.entries {
		if e.Meta.Path != path {
			kept = append(kept, e)
		}
	}
	idx.entries = kept
	if idx.sqlDB != nil {
		if _, err := idx.sqlDB.Exec(`DELETE FROM simhash_entries WHERE path=?`, path); err != nil {
			fmt.Printf("WARN: delete simhash by path: %v\n", err)
		}
	}
}

// FindSimilar returns entries within threshold of fp, excluding excludePath.
// Results are sorted by similarity descending, limited to topK.
func (idx *Index) FindSimilar(fp uint64, threshold float64, excludePath string, topK int) []Match {
	if topK <= 0 {
		topK = 5
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var matches []Match
	for _, e := range idx.entries {
		if e.Meta.Path == excludePath {
			continue
		}
		sim := Similarity(fp, e.Fingerprint)
		if sim >= threshold {
			matches = append(matches, Match{
				Meta:       e.Meta,
				Similarity: sim,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	if len(matches) > topK {
		matches = matches[:topK]
	}
	return matches
}

// Save persists the full index to SQLite.
func (idx *Index) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tx, err := idx.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM simhash_entries`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete simhash entries: %w", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO simhash_entries (fingerprint, project_id, path, symbol, start_line, end_line) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, e := range idx.entries {
		if _, err := stmt.Exec(int64(e.Fingerprint), e.Meta.ProjectID, e.Meta.Path, e.Meta.Symbol, e.Meta.StartLine, e.Meta.EndLine); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert simhash entry: %w", err)
		}
	}
	return tx.Commit()
}

func (idx *Index) load() error {
	rows, err := idx.sqlDB.Query(`SELECT fingerprint, project_id, path, symbol, start_line, end_line FROM simhash_entries`)
	if err != nil {
		return fmt.Errorf("query simhash entries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var fp int64
		var m Meta
		if err := rows.Scan(&fp, &m.ProjectID, &m.Path, &m.Symbol, &m.StartLine, &m.EndLine); err != nil {
			return fmt.Errorf("scan simhash row: %w", err)
		}
		idx.entries = append(idx.entries, indexEntry{Fingerprint: uint64(fp), Meta: m})
	}
	return rows.Err()
}
