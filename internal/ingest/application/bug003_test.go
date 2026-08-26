package application

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/acme/media-watermark-fingerprinting/internal/ingest/domain"
	storageapp "github.com/acme/media-watermark-fingerprinting/internal/storage/application"
	storagedomain "github.com/acme/media-watermark-fingerprinting/internal/storage/domain"
)

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(1_700_000_000, 0) }

var errUploadSentinel = errors.New("upload sentinel")

type fakeRepo struct {
	data []byte
}

func (f *fakeRepo) Create(context.Context, domain.Upload) error { return nil }
func (f *fakeRepo) Get(context.Context, string) (domain.Upload, error) {
	return domain.Upload{}, nil
}
func (f *fakeRepo) Append(context.Context, string, int64, []byte) (domain.Upload, error) {
	return domain.Upload{}, nil
}
func (f *fakeRepo) Data(context.Context, string) ([]byte, error) {
	if f.data != nil {
		return f.data, nil
	}
	return nil, errUploadSentinel
}
func (f *fakeRepo) Update(context.Context, domain.Upload) error { return nil }

type fakeObjects struct{}

func (fakeObjects) Put(context.Context, string, io.Reader, int64) (storagedomain.Object, error) {
	return storagedomain.Object{}, nil
}
func (fakeObjects) Open(context.Context, string) (io.ReadCloser, storagedomain.Object, error) {
	return nil, storagedomain.Object{}, nil
}
func (fakeObjects) Delete(context.Context, string) error { return nil }
func (fakeObjects) Exists(context.Context, string) bool  { return false }

type putErrorObjects struct{}

func (putErrorObjects) Put(context.Context, string, io.Reader, int64) (storagedomain.Object, error) {
	return storagedomain.Object{}, errUploadSentinel
}
func (putErrorObjects) Open(context.Context, string) (io.ReadCloser, storagedomain.Object, error) {
	return nil, storagedomain.Object{}, nil
}
func (putErrorObjects) Delete(context.Context, string) error { return nil }
func (putErrorObjects) Exists(context.Context, string) bool  { return false }

func TestIngestCompletePreservesStoreError(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, storageapp.New(fakeObjects{}, 1024), fakeClock{}, 1024)
	u := domain.Upload{ID: "u", Name: "u", ExpectedSize: 1, ReceivedSize: 1, Status: domain.StatusOpen}
	_, err := s.complete(context.Background(), u)
	if !errors.Is(err, errUploadSentinel) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}

func TestIngestCompletePreservesObjectPutError(t *testing.T) {
	repo := &fakeRepo{data: []byte("x")}
	objects := storageapp.New(putErrorObjects{}, 1024)
	s := New(repo, objects, fakeClock{}, 1024)
	u := domain.Upload{ID: "u", Name: "u", ExpectedSize: 1, ReceivedSize: 1, Status: domain.StatusOpen}
	_, err := s.complete(context.Background(), u)
	if !errors.Is(err, errUploadSentinel) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}
