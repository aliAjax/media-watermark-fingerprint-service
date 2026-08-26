package infrastructure

import (
	"context"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
)

func TestMemorySaveReviewInitializesMap(t *testing.T) {
	m := NewMemory()
	if err := m.SaveReview(context.Background(), domain.Review{ID: "r1", AssetID: "a1"}); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}
}

func TestMemoryCreateAssetInitializesMap(t *testing.T) {
	m := NewMemory()
	if err := m.CreateAsset(context.Background(), domain.Asset{ID: "a1", ObjectKey: "o1"}); err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
}

func TestMemoryAddEventInitializesMap(t *testing.T) {
	m := NewMemory()
	if err := m.AddEvent(context.Background(), domain.Event{ID: "e1", AssetID: "a1"}); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}
}

func TestMemoryUpdateAssetInitializesMap(t *testing.T) {
	m := NewMemory()
	_ = m.CreateAsset(context.Background(), domain.Asset{ID: "a1", ObjectKey: "o1"})
	if err := m.UpdateAsset(context.Background(), domain.Asset{ID: "a1", ObjectKey: "o2"}); err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
}
