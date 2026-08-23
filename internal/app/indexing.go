package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/directory"
	"github.com/quonaro/gnostis/internal/embeddings"
	"github.com/quonaro/gnostis/internal/graph"
	"github.com/quonaro/gnostis/internal/indexer"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/project"
	"github.com/quonaro/gnostis/internal/simhash"
	"github.com/quonaro/gnostis/internal/stats"
	"github.com/quonaro/gnostis/internal/store"
	"github.com/quonaro/gnostis/internal/symbol"
	"github.com/schollz/progressbar/v2"
)

const maxIndexRetries = 3

var indexBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
}

func indexDirectoryWithRetry(ctx context.Context, out io.Writer, dir directory.Directory, proj project.Project, idx *indexer.Indexer, ch *chunker.Chunker, provider embeddings.Provider, st store.VectorStore, sym *symbol.Index, cg *graph.Graph, si *simhash.Index, cache map[string][]float32, prog *progress.Progress, indexingStats *stats.Stats) error {
	var lastErr error
	for attempt := 0; attempt < maxIndexRetries; attempt++ {
		err := indexDirectory(ctx, out, dir, proj, idx, ch, provider, st, sym, cg, si, cache, prog, indexingStats)
		if err == nil {
			return nil
		}
		lastErr = err
		if errors.Is(err, fs.ErrNotExist) {
			slog.WarnContext(ctx, "project directory does not exist, skipping", "project", proj.Name, "path", dir.Path)
			return err
		}
		if attempt < maxIndexRetries-1 {
			slog.WarnContext(ctx, "index attempt failed, retrying", "project", proj.Name, "attempt", attempt+1, "error", err, "backoff", indexBackoff[attempt])
			select {
			case <-time.After(indexBackoff[attempt]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return lastErr
}

func indexDirectory(ctx context.Context, out io.Writer, dir directory.Directory, proj project.Project, idx *indexer.Indexer, ch *chunker.Chunker, provider embeddings.Provider, st store.VectorStore, sym *symbol.Index, cg *graph.Graph, si *simhash.Index, cache map[string][]float32, prog *progress.Progress, indexingStats *stats.Stats) error {
	if prog != nil {
		_ = prog.Start(proj.Name, 0)
	}

	files, err := idx.Index(ctx, dir, proj)
	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	slog.InfoContext(ctx, "indexed files", "project", proj.Name, "count", len(files))

	if prog != nil {
		_ = prog.Start(proj.Name, len(files))
		_ = prog.SetPhase(progress.PhaseChunking)
	}

	var bar *progressbar.ProgressBar
	if out != nil {
		bar = progressbar.NewOptions(len(files),
			progressbar.OptionSetWriter(out),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription(fmt.Sprintf("chunking %s", proj.Name)),
			progressbar.OptionSetPredictTime(true),
		)
	}

	changed := chunkFilesParallel(ctx, files, ch, st, sym, cg, si, bar, prog)

	totalChunks := 0
	for _, fc := range changed {
		totalChunks += len(fc.chunks)
	}
	if totalChunks == 0 {
		if bar != nil {
			_ = bar.Finish()
		}
		if prog != nil {
			_ = prog.Done()
		}
		slog.InfoContext(ctx, "no chunks to embed", "project", proj.Name)
		updateStats(ctx, indexingStats, st, proj.Name, provider.ModelName())
		return nil
	}

	if prog != nil {
		_ = prog.SetPhase(progress.PhaseEmbedding)
		_ = prog.SetTotalChunks(totalChunks)
	}

	if bar != nil {
		_ = bar.Finish()
		bar = progressbar.NewOptions(totalChunks,
			progressbar.OptionSetWriter(out),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription(fmt.Sprintf("embedding %s", proj.Name)),
			progressbar.OptionShowIts(),
			progressbar.OptionSetPredictTime(true),
		)
	}

	storedChunks := 0
	for i := 0; i < len(changed); i += embedBatchFiles {
		end := i + embedBatchFiles
		if end > len(changed) {
			end = len(changed)
		}
		batch := changed[i:end]

		batchChunks := make([]chunker.Chunk, 0, 64)
		for _, fc := range batch {
			batchChunks = append(batchChunks, fc.chunks...)
		}
		if len(batchChunks) == 0 {
			continue
		}

		vectors, err := embedChunks(ctx, provider, batchChunks, cache, func(done int) {
			if bar != nil {
				_ = bar.Add(done)
			}
			if prog != nil {
				_ = prog.AddChunks(done)
			}
		})
		if err != nil {
			return fmt.Errorf("embed chunks: %w", err)
		}

		if err := st.AddChunks(ctx, batchChunks, vectors); err != nil {
			return fmt.Errorf("store chunks: %w", err)
		}
		storedChunks += len(batchChunks)
	}

	if bar != nil {
		_ = bar.Finish()
	}

	updateStats(ctx, indexingStats, st, proj.Name, provider.ModelName())

	if prog != nil {
		_ = prog.Done()
	}
	slog.InfoContext(ctx, "stored chunks", "project", proj.Name, "count", storedChunks)
	return nil
}

const embedBatchFiles = 100

func updateStats(ctx context.Context, indexingStats *stats.Stats, st store.VectorStore, projectID string, model string) {
	if indexingStats == nil {
		return
	}
	count, _ := st.CountByProject(ctx, projectID)
	_ = indexingStats.Update(projectID, count, model)
}

type fileChunks struct {
	file   indexer.FileInfo
	chunks []chunker.Chunk
}

func progressAdd(bar *progressbar.ProgressBar, prog *progress.Progress, n int, mu *sync.Mutex) {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if bar != nil {
		_ = bar.Add(n)
	}
	if prog != nil {
		_ = prog.AddFiles(n)
	}
}

func chunkFilesParallel(ctx context.Context, files []indexer.FileInfo, ch *chunker.Chunker, st store.VectorStore, sym *symbol.Index, cg *graph.Graph, si *simhash.Index, bar *progressbar.ProgressBar, prog *progress.Progress) []fileChunks {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var progressMu sync.Mutex
	var changed []fileChunks

	// graphOnly tracks files that are unchanged but missing from the call graph.
	// They need re-chunking to populate the graph without re-embedding/storing.
	type graphOnlyEntry struct {
		file   indexer.FileInfo
		chunks []chunker.Chunk
	}
	var graphOnly []graphOnlyEntry

	for _, f := range files {
		storedHash, err := st.GetFileHash(ctx, f.Path)
		if err != nil {
			slog.WarnContext(ctx, "lookup stored hash", "path", f.Path, "error", err)
			progressAdd(bar, prog, 1, &progressMu)
			continue
		}
		if storedHash == f.Hash {
			if cg.HasFile(f.Path) {
				progressAdd(bar, prog, 1, &progressMu)
				continue
			}
			// File is unchanged but missing from the call graph — re-chunk
			// to populate the graph without re-embedding or re-storing.
			wg.Add(1)
			sem <- struct{}{}
			go func(file indexer.FileInfo) {
				defer wg.Done()
				defer func() { <-sem }()
				defer progressAdd(bar, prog, 1, &progressMu)

				chunks, err := ch.ChunkFile(ctx, file)
				if err != nil {
					slog.WarnContext(ctx, "chunk file for graph backfill", "path", file.Path, "error", err)
					return
				}
				if len(chunks) == 0 {
					return
				}
				mu.Lock()
				graphOnly = append(graphOnly, graphOnlyEntry{file: file, chunks: chunks})
				mu.Unlock()
			}(f)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(file indexer.FileInfo) {
			defer wg.Done()
			defer func() { <-sem }()
			defer progressAdd(bar, prog, 1, &progressMu)

			chunks, err := ch.ChunkFile(ctx, file)
			if err != nil {
				slog.WarnContext(ctx, "chunk file", "path", file.Path, "error", err)
				return
			}
			if len(chunks) == 0 {
				return
			}
			mu.Lock()
			changed = append(changed, fileChunks{file: file, chunks: chunks})
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	paths := make([]string, len(changed))
	for i, fc := range changed {
		paths[i] = fc.file.Path
	}
	if err := st.DeleteByPaths(ctx, paths); err != nil {
		slog.WarnContext(ctx, "delete stale chunks", "error", err)
	}
	for _, fc := range changed {
		sym.RemoveByPath(fc.file.Path)
		cg.RemoveByPath(fc.file.Path)
		si.RemoveByPath(fc.file.Path)
		sym.AddChunks(chunksToSymbolChunks(fc.chunks))
		for _, c := range fc.chunks {
			cg.AddChunk(c)
		}
		simhashAddChunks(si, fc.chunks)
	}

	// Backfill call graph, symbol index, and simhash for unchanged files
	// that were missing from the graph. No embedding or store update needed.
	for _, goe := range graphOnly {
		sym.RemoveByPath(goe.file.Path)
		cg.RemoveByPath(goe.file.Path)
		si.RemoveByPath(goe.file.Path)
		sym.AddChunks(chunksToSymbolChunks(goe.chunks))
		for _, c := range goe.chunks {
			cg.AddChunk(c)
		}
		simhashAddChunks(si, goe.chunks)
	}

	return changed
}

func reindexFile(ctx context.Context, absPath string, dirs []directory.Directory, projects []project.Project, st store.VectorStore, sym *symbol.Index, cg *graph.Graph, si *simhash.Index, provider embeddings.Provider, cache map[string][]float32, indexingStats *stats.Stats) error {
	if len(dirs) != len(projects) {
		return fmt.Errorf("directory and project count mismatch")
	}

	for i, dir := range dirs {
		if !strings.HasPrefix(absPath, dir.Path) {
			continue
		}
		return reindexFileUnder(ctx, absPath, dir, projects[i], st, sym, cg, si, provider, cache, indexingStats)
	}

	// Path is not under any configured directory. Index it under a synthetic
	// directory rooted at the file's parent so global rules still apply.
	parent := filepath.Dir(absPath)
	dir := directory.FromConfig(config.Directory{Path: parent, Name: filepath.Base(parent)})
	proj := project.New(filepath.Base(parent), parent)
	return reindexFileUnder(ctx, absPath, dir, proj, st, sym, cg, si, provider, cache, indexingStats)
}

func reindexFileUnder(ctx context.Context, absPath string, dir directory.Directory, proj project.Project, st store.VectorStore, sym *symbol.Index, cg *graph.Graph, si *simhash.Index, provider embeddings.Provider, cache map[string][]float32, indexingStats *stats.Stats) (err error) {
	defer func() {
		if err == nil {
			updateStats(ctx, indexingStats, st, proj.ID, provider.ModelName())
		}
	}()

	rel, err := filepath.Rel(dir.Path, absPath)
	if err != nil {
		return fmt.Errorf("relative path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		_ = st.DeleteByPath(ctx, absPath)
		sym.RemoveByPath(absPath)
		return nil
	}

	if info.IsDir() || !dir.ShouldIndex(rel, info.Size()) {
		_ = st.DeleteByPath(ctx, absPath)
		sym.RemoveByPath(absPath)
		return nil
	}

	slog.InfoContext(ctx, "reindexing file", "path", absPath, "project", proj.Name)

	_ = st.DeleteByPath(ctx, absPath)
	sym.RemoveByPath(absPath)
	cg.RemoveByPath(absPath)
	si.RemoveByPath(absPath)

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	f := indexer.FileInfo{
		ProjectID: proj.ID,
		Path:      absPath,
		RelPath:   rel,
		Content:   string(content),
		ModTime:   info.ModTime(),
	}

	ch := chunker.New()
	chunks, err := ch.ChunkFile(ctx, f)
	if err != nil {
		return fmt.Errorf("chunk file: %w", err)
	}
	if len(chunks) == 0 {
		return nil
	}

	sym.AddChunks(chunksToSymbolChunks(chunks))
	for _, c := range chunks {
		cg.AddChunk(c)
	}
	simhashAddChunks(si, chunks)

	vectors, err := embedChunks(ctx, provider, chunks, cache, nil)
	if err != nil {
		return fmt.Errorf("embed chunks: %w", err)
	}

	return st.AddChunks(ctx, chunks, vectors)
}
