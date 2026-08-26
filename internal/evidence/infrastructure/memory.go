package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
	"sync"
)

type Memory struct {
	mu      sync.RWMutex
	assets  map[string]domain.Asset
	events  map[string][]domain.Event
	reviews map[string]domain.Review
}

func NewMemory() *Memory {
	return &Memory{assets: make(map[string]domain.Asset), events: make(map[string][]domain.Event), reviews: make(map[string]domain.Review)}
}
func (m *Memory) CreateAsset(ctx context.Context, a domain.Asset) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.assets[a.ID]; ok {
		return fmt.Errorf("asset %q already exists", a.ID)
	}
	m.assets[a.ID] = a
	return nil
}
func (m *Memory) GetAsset(ctx context.Context, id string) (domain.Asset, error) {
	select {
	case <-ctx.Done():
		return domain.Asset{}, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assets[id]
	if !ok {
		return domain.Asset{}, fmt.Errorf("asset %q not found", id)
	}
	return a, nil
}
func (m *Memory) UpdateAsset(ctx context.Context, a domain.Asset) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.assets[a.ID]; !ok {
		return fmt.Errorf("asset %q not found", a.ID)
	}
	m.assets[a.ID] = a
	return nil
}
func (m *Memory) AddEvent(ctx context.Context, e domain.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[e.AssetID] = append(m.events[e.AssetID], e)
	return nil
}
func (m *Memory) Events(ctx context.Context, id string) ([]domain.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := append([]domain.Event(nil), m.events[id]...)
	return out, nil
}
func (m *Memory) SaveReview(ctx context.Context, r domain.Review) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.reviews[r.ID]; ok {
		return fmt.Errorf("review %q already exists", r.ID)
	}
	m.reviews[r.ID] = r
	return nil
}
