package chunker

import (
	"strings"

	"github.com/quonaro/gnostis/internal/simhash"
)

// maxChunkChars is the maximum number of characters per chunk.
// Most embedding models accept up to ~128K characters, but we use a
// conservative limit to stay well below common provider thresholds.
const maxChunkChars = 100_000

// splitLargeChunks breaks any chunk whose Content exceeds maxChunkChars
// into smaller line-based sub-chunks.
func splitLargeChunks(chunks []Chunk) []Chunk {
	out := make([]Chunk, 0, len(chunks))
	for _, c := range chunks {
		if len(c.Content) <= maxChunkChars {
			out = append(out, c)
			continue
		}
		out = append(out, splitChunk(c)...)
	}
	return out
}

// splitChunk divides a single oversized chunk into multiple chunks by lines.
func splitChunk(c Chunk) []Chunk {
	lines := strings.Split(c.Content, "\n")
	var result []Chunk

	startLine := c.StartLine
	var buf strings.Builder
	bufLen := 0
	sectionStart := startLine
	lineIdx := 0

	for _, line := range lines {
		lineLen := len(line) + 1 // +1 for newline
		if bufLen+lineLen > maxChunkChars && bufLen > 0 {
			text := strings.TrimSuffix(buf.String(), "\n")
			result = append(result, subChunk(c, text, sectionStart, lineIdx+startLine-1))
			buf.Reset()
			bufLen = 0
			sectionStart = lineIdx + startLine
		}
		buf.WriteString(line)
		buf.WriteString("\n")
		bufLen += lineLen
		lineIdx++
	}

	if bufLen > 0 {
		text := strings.TrimSuffix(buf.String(), "\n")
		result = append(result, subChunk(c, text, sectionStart, lineIdx+startLine-1))
	}

	return result
}

func subChunk(c Chunk, text string, startLine, endLine int) Chunk {
	return Chunk{
		ID:        hashChunk(c.Path, text),
		ProjectID: c.ProjectID,
		Path:      c.Path,
		FileHash:  c.FileHash,
		Language:  c.Language,
		Symbol:    c.Symbol,
		Signature: c.Signature,
		Docstring: c.Docstring,
		Content:   text,
		StartLine: startLine,
		EndLine:   endLine,
		Kind:      c.Kind,
		Calls:     c.Calls,
		Simhash:   simhash.Fingerprint(text),
	}
}
