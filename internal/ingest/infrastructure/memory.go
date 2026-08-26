package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/ingest/domain"
	"sync"
	"time"
)

type record struct {
	upload domain.Upload
	data   []byte
}
type Memory struct {
	mu      sync.RWMutex
	records map[string]record
}

func NewMemory() *Memory { return &Memory{records: make(map[string]record)} }
func (m *Memory) Create(ctx context.Context, u domain.Upload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.records[u.ID]; ok {
		return fmt.Errorf("upload %q exists", u.ID)
	}
	m.records[u.ID] = record{upload: u}
	return nil
}
func (m *Memory) Get(ctx context.Context, id string) (domain.Upload, error) {
	select {
	case <-ctx.Done():
		return domain.Upload{}, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.records[id]
	if !ok {
		return domain.Upload{}, fmt.Errorf("upload %q not found", id)
	}
	return r.upload, nil
}
func (m *Memory) Append(ctx context.Context, id string, offset int64, data []byte) (domain.Upload, error) {
	select {
	case <-ctx.Done():
		return domain.Upload{}, ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[id]
	if !ok {
		return domain.Upload{}, fmt.Errorf("upload %q not found", id)
	}
	if int64(len(r.data)) != offset {
		return r.upload, fmt.Errorf("offset conflict")
	}
	r.data = append(r.data, data...)
	r.upload.ReceivedSize = int64(len(r.data))
	r.upload.UpdatedAt = time.Now().UTC()
	m.records[id] = r
	return r.upload, nil
}
func (m *Memory) Data(ctx context.Context, id string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.records[id]
	if !ok {
		return nil, fmt.Errorf("upload %q not found", id)
	}
	return append([]byte(nil), r.data...), nil
}
func (m *Memory) Update(ctx context.Context, u domain.Upload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[u.ID]
	if !ok {
		return fmt.Errorf("upload %q not found", u.ID)
	}
	r.upload = u
	m.records[u.ID] = r
	return nil
}
