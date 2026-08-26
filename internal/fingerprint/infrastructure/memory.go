package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	"sync"
)

type Memory struct {
	mu    sync.RWMutex
	items map[string]domain.Fingerprint
}

func NewMemory() *Memory { return &Memory{items: make(map[string]domain.Fingerprint)} }
func (m *Memory) Save(ctx context.Context, f domain.Fingerprint) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if f.ID == "" || f.AssetID == "" {
		return fmt.Errorf("fingerprint id and asset id required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.items[f.ID]; exists {
		return fmt.Errorf("fingerprint %q already exists", f.ID)
	}
	f.Segments = append([]domain.Segment(nil), f.Segments...)
	m.items[f.ID] = f
	return nil
}
func (m *Memory) Get(ctx context.Context, id string) (domain.Fingerprint, error) {
	select {
	case <-ctx.Done():
		return domain.Fingerprint{}, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.items[id]
	if !ok {
		return domain.Fingerprint{}, fmt.Errorf("fingerprint %q not found", id)
	}
	f.Segments = append([]domain.Segment(nil), f.Segments...)
	return f, nil
}
func (m *Memory) List(ctx context.Context) ([]domain.Fingerprint, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Fingerprint, 0, len(m.items))
	for _, f := range m.items {
		f.Segments = append([]domain.Segment(nil), f.Segments...)
		out = append(out, f)
	}
	return out, nil
}
