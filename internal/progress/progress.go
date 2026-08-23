package progress

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/quonaro/gnostis/internal/db"
)

// Status values for a rebuild operation.
const (
	StatusIdle    = "idle"
	StatusRunning = "running"
	StatusError   = "error"
	StatusDone    = "done"
)

// Phase values describe the current stage of a rebuild.
const (
	PhaseIdle      = ""
	PhaseIndexing  = "indexing"
	PhaseChunking  = "chunking"
	PhaseEmbedding = "embedding"
)

// State describes the progress of an index rebuild.
type State struct {
	JobID       string    `json:"job_id,omitempty"`
	Status      string    `json:"status"`
	Phase       string    `json:"phase"`
	Project     string    `json:"project"`
	TotalFiles  int       `json:"total_files"`
	DoneFiles   int       `json:"done_files"`
	TotalChunks int       `json:"total_chunks"`
	DoneChunks  int       `json:"done_chunks"`
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Error       string    `json:"error,omitempty"`
}

// ETA estimates the remaining time based on the current processing rate.
// It returns 0 when the job is not running, no chunks have been processed,
// or all chunks are already done.
func (s State) ETA() time.Duration {
	if s.Status != StatusRunning || s.DoneChunks <= 0 || s.TotalChunks <= 0 || s.DoneChunks >= s.TotalChunks {
		return 0
	}
	elapsed := time.Now().UTC().Sub(s.StartedAt)
	if elapsed <= 0 {
		return 0
	}
	rate := float64(s.DoneChunks) / elapsed.Seconds()
	remaining := float64(s.TotalChunks-s.DoneChunks) / rate
	seconds := int64(remaining)
	if seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// Progress persists rebuild progress to SQLite.
type Progress struct {
	mu     sync.Mutex
	reader *sql.DB
	writer *sql.DB
	jobID  string
	state  State
}

// New creates a Progress writer backed by SQLite.
func New(database *db.DB) *Progress {
	return &Progress{
		reader: database.Reader(),
		writer: database.Writer(),
	}
}

// Load reads the persisted state from SQLite. If no row exists, it
// returns an idle state.
func (p *Progress) Load() (State, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var s State
	var jobID, startedAt, updatedAt, phase, project, errMsg sql.NullString
	var totalFiles, doneFiles, totalChunks, doneChunks, pid sql.NullInt64
	err := p.reader.QueryRow(`SELECT job_id, status, phase, project, total_files, done_files, total_chunks, done_chunks, pid, started_at, updated_at, error FROM progress_state WHERE id=1`).Scan(
		&jobID, &s.Status, &phase, &project, &totalFiles, &doneFiles, &totalChunks, &doneChunks, &pid, &startedAt, &updatedAt, &errMsg)
	if err == sql.ErrNoRows {
		p.state = State{Status: StatusIdle}
		return p.state, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("query progress: %w", err)
	}
	s.JobID = jobID.String
	s.Phase = phase.String
	s.Project = project.String
	s.Error = errMsg.String
	s.TotalFiles = int(totalFiles.Int64)
	s.DoneFiles = int(doneFiles.Int64)
	s.TotalChunks = int(totalChunks.Int64)
	s.DoneChunks = int(doneChunks.Int64)
	s.PID = int(pid.Int64)
	s.StartedAt, _ = time.Parse(time.RFC3339, startedAt.String)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	if s.JobID != "" {
		p.jobID = s.JobID
	}
	p.state = s
	if p.jobID != "" {
		p.state.JobID = p.jobID
	}
	return p.state, nil
}

// SetJobID sets the identifier of the current rebuild job.
// It is preserved across Start/SetPhase/AddFiles calls.
func (p *Progress) SetJobID(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobID = id
	p.state.JobID = id
	_ = p.saveLocked()
}

// Start resets the state for a new rebuild of the given project.
func (p *Progress) Start(project string, totalFiles int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now().UTC()
	p.state = State{
		JobID:      p.jobID,
		Status:     StatusRunning,
		Phase:      PhaseIndexing,
		Project:    project,
		TotalFiles: totalFiles,
		PID:        os.Getpid(),
		StartedAt:  now,
		UpdatedAt:  now,
	}
	return p.saveLocked()
}

// SetPhase updates the current phase.
func (p *Progress) SetPhase(phase string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.Phase = phase
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// SetTotalChunks updates the total number of chunks to embed.
func (p *Progress) SetTotalChunks(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.TotalChunks = n
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// AddFiles increments the number of processed files.
func (p *Progress) AddFiles(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.DoneFiles += n
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// AddChunks increments the number of embedded chunks.
func (p *Progress) AddChunks(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.DoneChunks += n
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// Reset clears any running/error state and returns to idle.
func (p *Progress) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = State{Status: StatusIdle, UpdatedAt: time.Now().UTC()}
	return p.saveLocked()
}

// Done marks the rebuild as successfully completed.
func (p *Progress) Done() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.Status = StatusDone
	p.state.Phase = PhaseIdle
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// Fail marks the rebuild as failed with the given error.
func (p *Progress) Fail(err error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.Status = StatusError
	if err != nil {
		p.state.Error = err.Error()
	}
	p.state.UpdatedAt = time.Now().UTC()
	return p.saveLocked()
}

// State returns a copy of the current state.
func (p *Progress) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Progress) saveLocked() error {
	if p.writer == nil {
		return nil
	}

	s := p.state
	_, err := p.writer.Exec(`INSERT OR REPLACE INTO progress_state (id, job_id, status, phase, project, total_files, done_files, total_chunks, done_chunks, pid, started_at, updated_at, error) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.JobID, s.Status, s.Phase, s.Project, s.TotalFiles, s.DoneFiles, s.TotalChunks, s.DoneChunks, s.PID, s.StartedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339), s.Error)
	if err != nil {
		return fmt.Errorf("upsert progress: %w", err)
	}
	return nil
}
