package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/storage/domain"
	"io"
	"time"
)

type ObjectStore interface {
	Put(context.Context, string, io.Reader, int64) (domain.Object, error)
	Open(context.Context, string) (io.ReadCloser, domain.Object, error)
	Delete(context.Context, string) error
	Exists(context.Context, string) bool
}
type Service struct {
	store    ObjectStore
	maxBytes int64
}

func New(store ObjectStore, maxBytes int64) *Service {
	return &Service{store: store, maxBytes: maxBytes}
}
func (s *Service) Put(ctx context.Context, key string, r io.Reader, size int64) (domain.Object, error) {
	if key == "" {
		return domain.Object{}, fmt.Errorf("key is required")
	}
	if size < 0 || size > s.maxBytes {
		return domain.Object{}, fmt.Errorf("object size %d exceeds limit", size)
	}
	return s.store.Put(ctx, key, io.LimitReader(r, s.maxBytes+1), size)
}
func (s *Service) Bytes(ctx context.Context, key string) ([]byte, domain.Object, error) {
	r, obj, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, obj, fmt.Errorf("open object: %w", err)
	}
	defer r.Close()
	var b bytes.Buffer
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(&b, h), io.LimitReader(r, s.maxBytes+1))
	if err != nil {
		return nil, obj, fmt.Errorf("read object: %w", err)
	}
	if n > s.maxBytes {
		return nil, obj, fmt.Errorf("object exceeds limit")
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if obj.SHA256 != "" && obj.SHA256 != digest {
		return nil, obj, fmt.Errorf("object checksum mismatch")
	}
	return b.Bytes(), obj, nil
}
func NewObject(key string, data []byte) domain.Object {
	h := sha256.Sum256(data)
	return domain.Object{Key: key, Size: int64(len(data)), SHA256: hex.EncodeToString(h[:]), CreatedAt: time.Now().UTC()}
}
