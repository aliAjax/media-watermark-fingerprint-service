package infrastructure

import (
	"context"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
)

func TestConfigStoreDraftNotCurrentUntilPublish(t *testing.T) {
	s := NewConfigStore()
	draft := domain.DefaultConfig()
	draft.VideoWindow = 12
	_, _ = s.CreateDraft(context.Background(), draft)
	cur, _ := s.Current(context.Background(), draft.ID)
	if cur.State != "published" || cur.VideoWindow != domain.DefaultConfig().VideoWindow {
		t.Fatalf("draft leaked into current: %+v", cur)
	}
}

func TestConfigStorePublishRetiresPreviousPublished(t *testing.T) {
	s := NewConfigStore()
	draft := domain.DefaultConfig()
	draft.VideoWindow = 12
	_, _ = s.CreateDraft(context.Background(), draft)
	_, _ = s.Publish(context.Background(), draft.ID)
	old := s.versions[draft.ID][0]
	if old.State != "retired" {
		t.Fatalf("previous published state = %s, want retired", old.State)
	}
}

func TestConfigStoreRollbackRestoresPublished(t *testing.T) {
	s := NewConfigStore()
	draft := domain.DefaultConfig()
	draft.VideoWindow = 12
	_, _ = s.CreateDraft(context.Background(), draft)
	_, _ = s.Publish(context.Background(), draft.ID)
	cur, _ := s.Rollback(context.Background(), draft.ID, 1)
	if cur.State != "published" || cur.PublishedAt == nil {
		t.Fatalf("rollback did not restore published version: %+v", cur)
	}
}
