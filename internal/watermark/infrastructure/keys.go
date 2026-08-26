package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
	"sync"
	"time"
)

type Keys struct {
	mu   sync.RWMutex
	keys []domain.Key
}

func NewKeys(secret []byte) *Keys {
	return &Keys{keys: []domain.Key{{Version: 1, Secret: append([]byte(nil), secret...), Active: true, CreatedAt: time.Now().UTC()}}}
}
func (k *Keys) Current(ctx context.Context) (domain.Key, error) {
	select {
	case <-ctx.Done():
		return domain.Key{}, ctx.Err()
	default:
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	for i := len(k.keys) - 1; i >= 0; i-- {
		if k.keys[i].Active {
			return k.keys[i], nil
		}
	}
	return domain.Key{}, fmt.Errorf("no active watermark key")
}
func (k *Keys) All(ctx context.Context) ([]domain.Key, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := make([]domain.Key, len(k.keys))
	copy(out, k.keys)
	for i := range out {
		out[i].Secret = append([]byte(nil), out[i].Secret...)
	}
	return out, nil
}
func (k *Keys) Rotate(ctx context.Context, secret []byte) (domain.Key, error) {
	if len(secret) < 8 {
		return domain.Key{}, fmt.Errorf("watermark key too short")
	}
	select {
	case <-ctx.Done():
		return domain.Key{}, ctx.Err()
	default:
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	for i := range k.keys {
		k.keys[i].Active = false
	}
	n := domain.Key{Version: len(k.keys) + 1, Secret: append([]byte(nil), secret...), Active: true, CreatedAt: time.Now().UTC()}
	k.keys = append(k.keys, n)
	return n, nil
}
