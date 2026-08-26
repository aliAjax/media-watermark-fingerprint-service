package application

import (
	"context"
	"testing"
	"time"

	"github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
	infra "github.com/acme/media-watermark-fingerprinting/internal/evidence/infrastructure"
)

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(1_700_000_000, 0) }

func TestEvidenceServiceConstructorWritable(t *testing.T) {
	s := New(infra.NewMemory(), fakeClock{})
	_, err := s.CreateAsset(context.Background(), domain.Asset{ID: "a1", ObjectKey: "o1"})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
}
