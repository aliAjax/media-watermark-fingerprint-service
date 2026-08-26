package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/storage/application"
	"github.com/acme/media-watermark-fingerprinting/internal/storage/domain"
	"io"
	"sync"
)

type stored struct {
	object domain.Object
	data   []byte
}
type Memory struct {
	mu      sync.RWMutex
	objects map[string]stored
}

func NewMemory() *Memory { return &Memory{objects: make(map[string]stored)} }
func (m *Memory) Put(ctx context.Context, key string, r io.Reader, size int64) (domain.Object, error) {
	select {
	case <-ctx.Done():
		return domain.Object{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return domain.Object{}, fmt.Errorf("read upload: %w", err)
	}
	if size >= 0 && int64(len(data)) != size {
		return domain.Object{}, fmt.Errorf("declared size %d differs from actual %d", size, len(data))
	}
	obj := application.NewObject(key, data)
	m.mu.Lock()
	m.objects[key] = stored{obj, append([]byte(nil), data...)}
	m.mu.Unlock()
	return obj, nil
}
func (m *Memory) Open(ctx context.Context, key string) (io.ReadCloser, domain.Object, error) {
	select {
	case <-ctx.Done():
		return nil, domain.Object{}, ctx.Err()
	default:
	}
	m.mu.RLock()
	v, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return nil, domain.Object{}, fmt.Errorf("object %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(v.data)), v.object, nil
}
func (m *Memory) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[key]; !ok {
		return fmt.Errorf("object %q not found", key)
	}
	delete(m.objects, key)
	return nil
}
func (m *Memory) Exists(ctx context.Context, key string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.objects[key]
	return ok
}
