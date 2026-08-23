package memory

import (
	"sync"
	"time"
)

// MemoryProgressStatus values for a memory indexing operation.
const (
	MemStatusIdle    = "idle"
	MemStatusRunning = "running"
	MemStatusDone    = "done"
	MemStatusError   = "error"
)

// ProgressState describes the progress of a memory (dialogue) indexing run.
type ProgressState struct {
	Status      string    `json:"status"`
	TotalFiles  int       `json:"total_files"`
	DoneFiles   int       `json:"done_files"`
	TotalChunks int       `json:"total_chunks"`
	DoneChunks  int       `json:"done_chunks"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Error       string    `json:"error,omitempty"`
}

// progressTracker is a thread-safe, in-memory progress tracker for memory
// indexing. Unlike the code index progress it is not persisted to disk because
// memory sync runs frequently and completes quickly.
type progressTracker struct {
	mu    sync.Mutex
	state ProgressState
}

func newProgressTracker() *progressTracker {
	return &progressTracker{}
}

func (t *progressTracker) start(totalFiles int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UTC()
	t.state = ProgressState{
		Status:     MemStatusRunning,
		TotalFiles: totalFiles,
		StartedAt:  now,
		UpdatedAt:  now,
	}
}

func (t *progressTracker) addFiles(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.DoneFiles += n
	t.state.UpdatedAt = time.Now().UTC()
}

func (t *progressTracker) done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Status = MemStatusDone
	t.state.UpdatedAt = time.Now().UTC()
}

func (t *progressTracker) fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Status = MemStatusError
	if err != nil {
		t.state.Error = err.Error()
	}
	t.state.UpdatedAt = time.Now().UTC()
}

func (t *progressTracker) snapshot() ProgressState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}
