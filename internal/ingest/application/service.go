package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/ingest/domain"
	storageapp "github.com/acme/media-watermark-fingerprinting/internal/storage/application"
	"time"
)

type Repository interface {
	Create(context.Context, domain.Upload) error
	Get(context.Context, string) (domain.Upload, error)
	Append(context.Context, string, int64, []byte) (domain.Upload, error)
	Data(context.Context, string) ([]byte, error)
	Update(context.Context, domain.Upload) error
}
type Clock interface{ Now() time.Time }
type Service struct {
	repo     Repository
	objects  *storageapp.Service
	clock    Clock
	maxBytes int64
}

func New(repo Repository, objects *storageapp.Service, clock Clock, maxBytes int64) *Service {
	return &Service{repo: repo, objects: objects, clock: clock, maxBytes: maxBytes}
}
func (s *Service) Start(ctx context.Context, id, name string, size int64) (domain.Upload, error) {
	if id == "" || name == "" {
		return domain.Upload{}, fmt.Errorf("upload id and name required")
	}
	if size < 1 || size > s.maxBytes {
		return domain.Upload{}, fmt.Errorf("expected size outside limit")
	}
	now := s.clock.Now()
	u := domain.Upload{ID: id, Name: name, ExpectedSize: size, Status: domain.StatusOpen, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, u); err != nil {
		return u, fmt.Errorf("create upload: %w", err)
	}
	return u, nil
}
func (s *Service) Append(ctx context.Context, id string, chunk domain.Chunk) (domain.Upload, error) {
	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return u, err
	}
	if u.Status != domain.StatusOpen {
		return u, fmt.Errorf("upload is not open")
	}
	if chunk.Offset != u.ReceivedSize {
		return u, fmt.Errorf("chunk offset %d does not match next offset %d", chunk.Offset, u.ReceivedSize)
	}
	if int64(len(chunk.Data))+u.ReceivedSize > u.ExpectedSize {
		return u, fmt.Errorf("chunk exceeds expected size")
	}
	u, err = s.repo.Append(ctx, id, chunk.Offset, chunk.Data)
	if err != nil {
		return u, fmt.Errorf("append chunk: %w", err)
	}
	if chunk.Final || u.ReceivedSize == u.ExpectedSize {
		return s.complete(ctx, u)
	}
	return u, nil
}
func (s *Service) complete(ctx context.Context, u domain.Upload) (domain.Upload, error) {
	if u.ReceivedSize != u.ExpectedSize {
		return u, fmt.Errorf("cannot finalize incomplete upload")
	}
	data, err := s.repo.Data(ctx, u.ID)
	if err != nil {
		return u, fmt.Errorf("read upload: %w", err)
	}
	sum := sha256.Sum256(data)
	u.SHA256 = hex.EncodeToString(sum[:])
	u.ObjectKey = "assets/" + u.ID + "/original"
	if _, err := s.objects.Put(ctx, u.ObjectKey, bytes.NewReader(data), int64(len(data))); err != nil {
		return u, fmt.Errorf("store upload: %w", err)
	}
	u.Status = domain.StatusComplete
	u.UpdatedAt = s.clock.Now()
	if err := s.repo.Update(ctx, u); err != nil {
		return u, fmt.Errorf("finish upload: %w", err)
	}
	return u, nil
}
func (s *Service) Get(ctx context.Context, id string) (domain.Upload, error) {
	return s.repo.Get(ctx, id)
}
