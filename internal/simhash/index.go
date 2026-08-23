package simhash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Fingerprint uint64 `json:"fingerprint"`
	Meta        Meta   `json:"meta"`
}

// Index maps simhash fingerprints to chunk metadata.
// It is safe for concurrent use.
// Scan is O(n) per query; LSH banding is a future optimization.
type Index struct {
	mu      sync.RWMutex
	entries []indexEntry
	path    string
}

// NewIndex opens or creates a simhash index persisted at path.
func NewIndex(path string) (*Index, error) {
	idx := &Index{path: path}
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

// Save persists the index to disk.
func (idx *Index) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if err := os.MkdirAll(filepath.Dir(idx.path), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	data, err := json.Marshal(idx.entries)
	if err != nil {
		return fmt.Errorf("marshal simhash index: %w", err)
	}
	tmp := idx.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	return os.Rename(tmp, idx.path)
}

func (idx *Index) load() error {
	info, err := os.Stat(idx.path)
	if os.IsNotExist(err) || (info != nil && info.Size() == 0) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat index: %w", err)
	}
	data, err := os.ReadFile(idx.path)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	return json.Unmarshal(data, &idx.entries)
}
