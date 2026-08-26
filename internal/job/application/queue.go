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
	if j.Status == domain.StatusQueued || j.Status == domain.StatusRunning {
		q.mu.RLock()
		cancel := q.cancels[id]
		q.mu.RUnlock()
		if cancel != nil {
			cancel()
		}
		j.Status = domain.StatusCanceled
		j.Error = "canceled by request"
		now := q.clock.Now()
		j.FinishedAt = &now
		if err := q.store.Update(ctx, j); err != nil {
			return j, err
		}
	}
	return j, nil
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
				j.Status = domain.StatusFailed
				j.Error = "no handler"
				_ = q.store.Update(context.Background(), j)
				continue
			}
			jobCtx, cancel := context.WithCancel(context.Background())
			q.mu.Lock()
			q.cancels[j.ID] = cancel
			q.mu.Unlock()
			j.Status = domain.StatusRunning
			j.Attempt++
			now := q.clock.Now()
			j.StartedAt = &now
			_ = q.store.Update(context.Background(), j)
			err := h(jobCtx, j)
			q.mu.Lock()
			delete(q.cancels, j.ID)
			q.mu.Unlock()
			cancel()
			latest, _ := q.store.Get(context.Background(), j.ID)
			if latest.Status == domain.StatusCanceled {
				did = true
				continue
			}
			if err == nil {
				j.Status = domain.StatusSucceeded
				j.Error = ""
			} else {
				j.Error = err.Error()
				if j.Attempt >= j.MaxAttempts {
					j.Status = domain.StatusDead
				} else {
					j.Status = domain.StatusQueued
				}
			}
			end := q.clock.Now()
			j.FinishedAt = &end
			_ = q.store.Update(context.Background(), j)
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
