package infrastructure

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	"sync"
	"time"
)

type ConfigStore struct {
	mu       sync.RWMutex
	versions map[string][]domain.AlgorithmConfig
}

func NewConfigStore() *ConfigStore {
	s := &ConfigStore{versions: make(map[string][]domain.AlgorithmConfig)}
	d := domain.DefaultConfig()
	now := time.Now().UTC()
	d.PublishedAt = &now
	s.versions[d.ID] = []domain.AlgorithmConfig{d}
	return s
}
func (s *ConfigStore) CreateDraft(ctx context.Context, c domain.AlgorithmConfig) (domain.AlgorithmConfig, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	select {
	case <-ctx.Done():
		return c, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.versions[c.ID]
	c.Version = len(list) + 1
	c.State = "draft"
	c.CreatedAt = time.Now().UTC()
	c.PublishedAt = nil
	s.versions[c.ID] = append(list, c)
	return c, nil
}
func (s *ConfigStore) Publish(ctx context.Context, id string) (domain.AlgorithmConfig, error) {
	select {
	case <-ctx.Done():
		return domain.AlgorithmConfig{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.versions[id]
	if len(list) == 0 {
		return domain.AlgorithmConfig{}, fmt.Errorf("algorithm %q not found", id)
	}
	index := -1
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].State == "draft" {
			index = i
			break
		}
	}
	if index < 0 {
		return domain.AlgorithmConfig{}, fmt.Errorf("algorithm %q has no draft", id)
	}
	now := time.Now().UTC()
	list[index].State = "published"
	list[index].PublishedAt = &now
	s.versions[id] = list
	return list[index], nil
}
func (s *ConfigStore) Rollback(ctx context.Context, id string, version int) (domain.AlgorithmConfig, error) {
	select {
	case <-ctx.Done():
		return domain.AlgorithmConfig{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.versions[id]
	target := -1
	for i := range list {
		if list[i].Version == version {
			target = i
		}
	}
	if target < 0 {
		return domain.AlgorithmConfig{}, fmt.Errorf("version %d not found", version)
	}
	for i := range list {
		if list[i].State == "published" {
			list[i].State = "retired"
		}
	}
	list[target].State = "published"
	list[target].PublishedAt = nil
	s.versions[id] = list
	return list[target], nil
}
func (s *ConfigStore) Current(ctx context.Context, id string) (domain.AlgorithmConfig, error) {
	select {
	case <-ctx.Done():
		return domain.AlgorithmConfig{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.versions[id]
	if len(list) > 0 {
		return list[len(list)-1], nil
	}
	return domain.AlgorithmConfig{}, fmt.Errorf("published algorithm %q not found", id)
}
