package infrastructure

import (
	"context"
	"sync"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/job/domain"
)

func TestMemoryGetRace(t *testing.T) {
	m := NewMemory()
	if err := m.Create(context.Background(), domain.Job{ID: "j1", Status: domain.StatusQueued}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = m.Get(context.Background(), "j1")
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_ = m.Update(context.Background(), domain.Job{ID: "j1", Status: domain.StatusRunning})
		}
	}()
	close(start)
	wg.Wait()
}
