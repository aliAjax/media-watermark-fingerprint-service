package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/storage/domain"
)

var errStoreSentinel = errors.New("store sentinel")

type failingStore struct{}

func (f failingStore) Put(context.Context, string, io.Reader, int64) (domain.Object, error) {
	return domain.Object{}, errStoreSentinel
}
func (f failingStore) Open(context.Context, string) (io.ReadCloser, domain.Object, error) {
	return nil, domain.Object{}, errStoreSentinel
}
func (f failingStore) Delete(context.Context, string) error { return nil }
func (f failingStore) Exists(context.Context, string) bool  { return false }

func TestStorageBytesPreservesOpenError(t *testing.T) {
	s := New(failingStore{}, 1024)
	_, _, err := s.Bytes(context.Background(), "missing")
	if !errors.Is(err, errStoreSentinel) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}

func TestStoragePutPreservesObjectError(t *testing.T) {
	s := New(failingStore{}, 1024)
	_, err := s.Put(context.Background(), "missing", strings.NewReader("x"), 1)
	if !errors.Is(err, errStoreSentinel) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}

type readErrorStore struct{}

func (readErrorStore) Put(context.Context, string, io.Reader, int64) (domain.Object, error) {
	return domain.Object{}, nil
}
func (readErrorStore) Open(context.Context, string) (io.ReadCloser, domain.Object, error) {
	return readErrorCloser{}, domain.Object{SHA256: "0000000000000000000000000000000000000000000000000000000000000000"}, nil
}
func (readErrorStore) Delete(context.Context, string) error { return nil }
func (readErrorStore) Exists(context.Context, string) bool  { return false }

type readErrorCloser struct{}

func (readErrorCloser) Read([]byte) (int, error) { return 0, errStoreSentinel }
func (readErrorCloser) Close() error            { return nil }

func TestStorageBytesPreservesReadError(t *testing.T) {
	s := New(readErrorStore{}, 1024)
	_, _, err := s.Bytes(context.Background(), "read-error")
	if !errors.Is(err, errStoreSentinel) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}
