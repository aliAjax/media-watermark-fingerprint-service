package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/job/domain"
	"sync"
)

type Memory struct {
	mu   sync.RWMutex
	jobs map[string]domain.Job
}

func NewMemory() *Memory { return &Memory{jobs: make(map[string]domain.Job)} }
func (m *Memory) Create(ctx context.Context, j domain.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[j.ID]; ok {
		return fmt.Errorf("job %q already exists", j.ID)
	}
	m.jobs[j.ID] = j
	return nil
}
func (m *Memory) Get(ctx context.Context, id string) (domain.Job, error) {
	select {
	case <-ctx.Done():
		return domain.Job{}, ctx.Err()
	default:
	}
	j, ok := m.jobs[id]
	if !ok {
		return domain.Job{}, fmt.Errorf("job %q not found", id)
	}
	return j, nil
}
func (m *Memory) Update(ctx context.Context, j domain.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[j.ID]; !ok {
		return fmt.Errorf("job %q not found", j.ID)
	}
	m.jobs[j.ID] = j
	return nil
}
func (m *Memory) List(ctx context.Context) ([]domain.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out, nil
}
