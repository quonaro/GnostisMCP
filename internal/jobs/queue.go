package jobs

import (
	"context"
	"sync"
	"time"
)

// Status represents the lifecycle state of a job.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// Job describes a unit of work in the queue.
type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`

	fn func(context.Context) error
}

// Queue holds pending, running, and recently completed jobs.
// Jobs execute sequentially in a single worker goroutine.
type Queue struct {
	mu         sync.Mutex
	jobs       []Job
	notify     chan struct{}
	maxHistory int
}

// New creates a queue that retains up to maxHistory completed jobs.
func New(maxHistory int) *Queue {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	return &Queue{
		notify:     make(chan struct{}, 1),
		maxHistory: maxHistory,
	}
}

// Submit adds a job to the queue and returns its ID.
func (q *Queue) Submit(jobType, description string, fn func(context.Context) error) string {
	id := generateID(jobType)
	q.SubmitWithID(id, jobType, description, fn)
	return id
}

// SubmitWithID adds a job with a specific ID (used for resuming interrupted jobs).
func (q *Queue) SubmitWithID(id, jobType, description string, fn func(context.Context) error) {
	q.mu.Lock()
	q.jobs = append(q.jobs, Job{
		ID:          id,
		Type:        jobType,
		Description: description,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		fn:          fn,
	})
	q.mu.Unlock()
	q.signal()
}

// Snapshot returns a copy of all jobs (pending, running, recently completed).
func (q *Queue) Snapshot() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Job, len(q.jobs))
	copy(out, q.jobs)
	for i := range out {
		out[i].fn = nil
	}
	return out
}

// RunningJobID returns the ID of the currently running job, or empty string.
func (q *Queue) RunningJobID() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].Status == StatusRunning {
			return q.jobs[i].ID
		}
	}
	return ""
}

// Start launches the worker goroutine. It blocks until ctx is cancelled.
func (q *Queue) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.notify:
			q.processNext(ctx)
		}
	}
}

func (q *Queue) processNext(ctx context.Context) {
	q.mu.Lock()
	idx := -1
	for i := range q.jobs {
		if q.jobs[i].Status == StatusPending {
			idx = i
			break
		}
	}
	if idx == -1 {
		q.mu.Unlock()
		return
	}

	now := time.Now()
	q.jobs[idx].Status = StatusRunning
	q.jobs[idx].StartedAt = &now
	job := q.jobs[idx]
	q.mu.Unlock()

	err := job.fn(context.WithoutCancel(ctx))

	q.mu.Lock()
	fin := time.Now()
	q.jobs[idx].FinishedAt = &fin
	if err != nil {
		q.jobs[idx].Status = StatusFailed
		q.jobs[idx].Error = err.Error()
	} else {
		q.jobs[idx].Status = StatusDone
	}
	q.jobs[idx].fn = nil
	q.prune()
	q.mu.Unlock()

	if err == nil {
		q.signal()
	}
}

// prune removes old completed jobs beyond maxHistory, keeping at most
// maxHistory non-pending/non-running entries.
func (q *Queue) prune() {
	if len(q.jobs) <= q.maxHistory {
		return
	}
	kept := make([]Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		if j.Status == StatusPending || j.Status == StatusRunning {
			kept = append(kept, j)
		}
	}
	for _, j := range q.jobs {
		if j.Status == StatusDone || j.Status == StatusFailed {
			if len(kept) >= q.maxHistory {
				break
			}
			kept = append(kept, j)
		}
	}
	q.jobs = kept
}

func (q *Queue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().UTC().Format("20060102150405.000000")
}
