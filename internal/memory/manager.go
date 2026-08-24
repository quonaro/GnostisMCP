package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quonaro/gnostis/internal/db"
	"github.com/quonaro/gnostis/internal/embeddings"
)

// Manager manages user-written memory notes stored as Markdown files
// and indexed in a vector store for semantic search.
type Manager struct {
	dataDir  string
	indexer  *Indexer
	provider embeddings.Provider
	cache    map[string][]float32
}

// NewManager creates a memory manager for user notes.
func NewManager(dataDir string, provider embeddings.Provider, database *db.DB) (*Manager, error) {
	store, err := NewStore(context.Background(), dataDir, database)
	if err != nil {
		return nil, fmt.Errorf("open memory store: %w", err)
	}

	return &Manager{
		dataDir:  dataDir,
		indexer:  NewIndexer(store),
		provider: provider,
		cache:    make(map[string][]float32),
	}, nil
}

// Start creates the memory data directory if it does not exist.
func (m *Manager) Start(_ context.Context) error {
	return os.MkdirAll(m.dataDir, 0o755)
}

// Stop is a no-op (no background goroutines to stop).
func (m *Manager) Stop() error { return nil }

// Rebuild clears the memory store and re-indexes all note files.
func (m *Manager) Rebuild(ctx context.Context) error {
	store := m.Store()
	paths := store.Paths()
	if err := store.DeleteByPaths(ctx, paths); err != nil {
		return fmt.Errorf("clear memory store: %w", err)
	}

	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return fmt.Errorf("read memory data dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(m.dataDir, entry.Name())
		if err := m.indexer.IndexFile(ctx, "note", path, m.provider, m.cache); err != nil {
			return fmt.Errorf("index memory file %s: %w", path, err)
		}
	}
	return nil
}

// Store returns the underlying memory store for use by MCP tools.
func (m *Manager) Store() *Store {
	return m.indexer.store
}

// Provider returns the embedding provider used by the memory manager.
func (m *Manager) Provider() embeddings.Provider {
	return m.provider
}

// DataDir returns the directory where memory Markdown files are stored.
func (m *Manager) DataDir() string {
	return m.dataDir
}

// WriteNote writes a user note to the memory data dir and indexes it.
func (m *Manager) WriteNote(ctx context.Context, title, content string, tags []string, providerID string) (string, error) {
	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return "", fmt.Errorf("create memory data dir: %w", err)
	}

	slug := slugify(title)
	if slug == "" {
		slug = "note"
	}
	filename := fmt.Sprintf("%s-%s.md", time.Now().UTC().Format("20060102-150405"), slug)
	path := filepath.Join(m.dataDir, filename)

	if err := writeNoteMarkdown(path, title, content, tags, providerID); err != nil {
		return "", fmt.Errorf("write note: %w", err)
	}

	if err := m.indexer.IndexFile(ctx, providerID, path, m.provider, m.cache); err != nil {
		return "", fmt.Errorf("index note: %w", err)
	}

	return path, nil
}

func writeNoteMarkdown(path, title, content string, tags []string, providerID string) error {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString("- **Provider:** ")
	b.WriteString(providerID)
	b.WriteString("\n")
	if len(tags) > 0 {
		b.WriteString("- **Tags:** ")
		b.WriteString(strings.Join(tags, ", "))
		b.WriteString("\n")
	}
	b.WriteString("- **Saved:** ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\n\n")
	b.WriteString(content)
	b.WriteString("\n")

	return os.WriteFile(path, []byte(b.String()), 0o600)
}

var slugRe = strings.NewReplacer(
	" ", "-",
	"_", "-",
	"/", "-",
	"\\", "-",
	":", "-",
)

func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugRe.Replace(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	s = strings.Trim(b.String(), "-")
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		s = "note"
	}
	return s
}
