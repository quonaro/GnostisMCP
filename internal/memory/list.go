package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds metadata about a single exported memory file.
type FileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Title    string `json:"title"`
	Source   string `json:"source,omitempty"`
	Date     string `json:"date,omitempty"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
}

// ListFiles returns metadata for every exported Markdown file in the memory
// data directory, sorted by modification time (newest first).
func (m *Manager) ListFiles(_ context.Context) []FileInfo {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return nil
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(m.dataDir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		meta := parseMemoryMetadata(string(content))

		files = append(files, FileInfo{
			Path:     path,
			Name:     entry.Name(),
			Provider: meta.provider,
			Title:    meta.title,
			Source:   meta.source,
			Date:     meta.date,
			Size:     info.Size(),
			Type:     meta.fileType,
		})
	}

	sortFilesNewestFirst(files)
	return files
}

type memoryMetadata struct {
	title    string
	provider string
	source   string
	date     string
	fileType string
}

func parseMemoryMetadata(content string) memoryMetadata {
	var meta memoryMetadata

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)

		if meta.title == "" && strings.HasPrefix(line, "# ") {
			meta.title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if strings.HasPrefix(meta.title, "Chat trajectory:") {
				meta.fileType = "chat"
				meta.title = strings.TrimSpace(strings.TrimPrefix(meta.title, "Chat trajectory:"))
			} else {
				meta.fileType = "note"
			}
			continue
		}

		if strings.HasPrefix(line, "- **Provider:**") {
			meta.provider = strings.TrimSpace(strings.TrimPrefix(line, "- **Provider:**"))
			continue
		}

		if strings.HasPrefix(line, "- **Source:**") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- **Source:**"))
			meta.source = strings.Trim(val, "`")
			continue
		}

		if strings.HasPrefix(line, "- **Exported:**") {
			meta.date = strings.TrimSpace(strings.TrimPrefix(line, "- **Exported:**"))
			continue
		}

		if strings.HasPrefix(line, "- **Saved:**") {
			meta.date = strings.TrimSpace(strings.TrimPrefix(line, "- **Saved:**"))
			continue
		}
	}

	if meta.provider == "" {
		meta.provider = "unknown"
	}
	if meta.title == "" {
		meta.title = "Untitled"
	}

	return meta
}

func sortFilesNewestFirst(files []FileInfo) {
	for i := 1; i < len(files); i++ {
		for j := i; j > 0 && files[j].Date > files[j-1].Date; j-- {
			files[j], files[j-1] = files[j-1], files[j]
		}
	}
}
