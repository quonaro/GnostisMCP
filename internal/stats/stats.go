package stats

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/quonaro/gnostis/internal/db"
)

// Project holds per-project indexing metadata.
type Project struct {
	Path          string    `json:"path"`
	Chunks        int       `json:"chunks"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
	Model         string    `json:"model,omitempty"`
	Extensions    []string  `json:"extensions,omitempty"`
	Include       []string  `json:"include,omitempty"`
	Exclude       []string  `json:"exclude,omitempty"`
	MaxFileSizeMB int       `json:"max_file_size_mb,omitempty"`
}

// Stats persists per-project indexing statistics.
type Stats struct {
	mu     sync.Mutex
	reader *sql.DB
	writer *sql.DB
	data   map[string]Project
}

// New creates a Stats writer backed by SQLite.
func New(database *db.DB) *Stats {
	return &Stats{
		reader: database.Reader(),
		writer: database.Writer(),
		data:   make(map[string]Project),
	}
}

// Load reads the persisted stats from SQLite.
func (s *Stats) Load() (map[string]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.reader.Query(`SELECT project, path, chunks, last_indexed_at, model FROM project_stats`)
	if err != nil {
		return nil, fmt.Errorf("query project stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var p Project
		var project, lastIndexedAt string
		if err := rows.Scan(&project, &p.Path, &p.Chunks, &lastIndexedAt, &p.Model); err != nil {
			return nil, fmt.Errorf("scan stats row: %w", err)
		}
		p.LastIndexedAt, _ = time.Parse(time.RFC3339Nano, lastIndexedAt)
		s.data[project] = p
	}
	return s.data, rows.Err()
}

// Update records the chunk count, embedding model, and current time for the
// given project.
func (s *Stats) Update(project string, chunks int, model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.data[project] = Project{
		Chunks:        chunks,
		Model:         model,
		LastIndexedAt: now,
	}
	_, err := s.writer.Exec(`INSERT OR REPLACE INTO project_stats (project, path, chunks, last_indexed_at, model) VALUES (?, ?, ?, ?, ?)`,
		project, s.data[project].Path, chunks, now.Format(time.RFC3339Nano), model)
	if err != nil {
		return fmt.Errorf("upsert project stats: %w", err)
	}
	return nil
}
