package application

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/job/domain"
	"sort"
	"sync"
	"time"
)

type Store interface {
	Create(context.Context, domain.Job) error
	Get(context.Context, string) (domain.Job, error)
	Update(context.Context, domain.Job) error
	List(context.Context) ([]domain.Job, error)
	CompareAndSet(ctx context.Context, id string, expected map[domain.Status]struct{}, apply func(domain.Job) domain.Job) (domain.Job, bool, error)
}
type Clock interface{ Now() time.Time }
type Queue struct {
	store       Store
	clock       Clock
	handlers    map[string]domain.Handler
	mu          sync.RWMutex
	cancels     map[string]context.CancelFunc
	wake        chan struct{}
	closed      chan struct{}
	done        chan struct{}
	maxAttempts int
}

func NewQueue(store Store, clock Clock, workers int, maxAttempts int) *Queue {
	if workers < 1 {
		workers = 1
	}
	if maxAttempts < 1 {
		maxAttempts = 3
	}
	q := &Queue{store: store, clock: clock, handlers: make(map[string]domain.Handler), cancels: make(map[string]context.CancelFunc), wake: make(chan struct{}, 1), closed: make(chan struct{}), done: make(chan struct{}), maxAttempts: maxAttempts}
	go q.run(workers)
	return q
}
func (q *Queue) Register(kind string, h domain.Handler) {
	q.mu.Lock()
	q.handlers[kind] = h
	q.mu.Unlock()
}
func (q *Queue) Enqueue(ctx context.Context, j domain.Job) (domain.Job, error) {
	if j.ID == "" || j.Kind == "" {
		return j, fmt.Errorf("job id and kind required")
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = q.maxAttempts
	}
	j.Status = domain.StatusQueued
	j.CreatedAt = q.clock.Now()
	if err := q.store.Create(ctx, j); err != nil {
		return j, fmt.Errorf("enqueue job: %w", err)
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return j, nil
}
func (q *Queue) Get(ctx context.Context, id string) (domain.Job, error) { return q.store.Get(ctx, id) }
func (q *Queue) Cancel(ctx context.Context, id string) (domain.Job, error) {
	j, err := q.store.Get(ctx, id)
	if err != nil {
		return j, err
	}
	if j.Status != domain.StatusQueued && j.Status != domain.StatusRunning {
		return j, nil
	}
	q.mu.RLock()
	cancel := q.cancels[id]
	q.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
	updated, ok, err := q.store.CompareAndSet(ctx, id, statusSet(domain.StatusQueued, domain.StatusRunning), func(cur domain.Job) domain.Job {
		cur.Status = domain.StatusCanceled
		cur.Error = "canceled by request"
		now := q.clock.Now()
		cur.FinishedAt = &now
		return cur
	})
	if err != nil {
		return j, err
	}
	if ok {
		return updated, nil
	}
	// The job transitioned out of queued/running before we could cancel it
	// (e.g. a worker finished it concurrently). Return the current state.
	cur, err := q.store.Get(ctx, id)
	if err != nil {
		return j, nil
	}
	return cur, nil
}
func (q *Queue) run(workers int) {
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); q.worker() }()
	}
	wg.Wait()
	close(q.done)
}
func statusSet(statuses ...domain.Status) map[domain.Status]struct{} {
	out := make(map[domain.Status]struct{}, len(statuses))
	for _, s := range statuses {
		out[s] = struct{}{}
	}
	return out
}
func (q *Queue) worker() {
	for {
		select {
		case <-q.closed:
			return
		default:
		}
		jobs, _ := q.store.List(context.Background())
		sort.Slice(jobs, func(i, j int) bool {
			if jobs[i].Priority == jobs[j].Priority {
				return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
			}
			return jobs[i].Priority > jobs[j].Priority
		})
		did := false
		for _, j := range jobs {
			if j.Status != domain.StatusQueued {
				continue
			}
			q.mu.RLock()
			h := q.handlers[j.Kind]
			q.mu.RUnlock()
			if h == nil {
				_, _, _ = q.store.CompareAndSet(context.Background(), j.ID, statusSet(domain.StatusQueued), func(cur domain.Job) domain.Job {
					cur.Status = domain.StatusFailed
					cur.Error = "no handler"
					now := q.clock.Now()
					cur.FinishedAt = &now
					return cur
				})
				continue
			}
			// Atomically claim the job: only transition queued -> running.
			// This prevents two workers from picking up the same job.
			claimed, ok, _ := q.store.CompareAndSet(context.Background(), j.ID, statusSet(domain.StatusQueued), func(cur domain.Job) domain.Job {
				cur.Status = domain.StatusRunning
				cur.Attempt++
				now := q.clock.Now()
				cur.StartedAt = &now
				return cur
			})
			if !ok {
				continue
			}
			j = claimed
			jobCtx, cancel := context.WithCancel(context.Background())
			q.mu.Lock()
			q.cancels[j.ID] = cancel
			q.mu.Unlock()
			err := h(jobCtx, j)
			q.mu.Lock()
			delete(q.cancels, j.ID)
			q.mu.Unlock()
			cancel()
			// Finalize only if the job is still running. If a cancel raced in
			// and moved it to canceled, leave that terminal state intact.
			_, _, _ = q.store.CompareAndSet(context.Background(), j.ID, statusSet(domain.StatusRunning), func(cur domain.Job) domain.Job {
				if err == nil {
					cur.Status = domain.StatusSucceeded
					cur.Error = ""
				} else {
					cur.Error = err.Error()
					if cur.Attempt >= cur.MaxAttempts {
						cur.Status = domain.StatusDead
					} else {
						cur.Status = domain.StatusQueued
					}
				}
				end := q.clock.Now()
				cur.FinishedAt = &end
				return cur
			})
			did = true
		}
		if !did {
			select {
			case <-q.wake:
			case <-time.After(100 * time.Millisecond):
			case <-q.closed:
				return
			}
		}
	}
}
func (q *Queue) Shutdown(ctx context.Context) error {
	close(q.closed)
	select {
	case <-q.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
