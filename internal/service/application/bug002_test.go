package application

import (
	"context"
	"testing"
	"time"

	jobapp "github.com/acme/media-watermark-fingerprinting/internal/job/application"
	jobdomain "github.com/acme/media-watermark-fingerprinting/internal/job/domain"
	jobmemory "github.com/acme/media-watermark-fingerprinting/internal/job/infrastructure"
	workerapp "github.com/acme/media-watermark-fingerprinting/internal/worker/application"
)

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

func TestAnalysisHandlerCancellationReleasesSlot(t *testing.T) {
	store := jobmemory.NewMemory()
	q := jobapp.NewQueue(store, fakeClock{now: time.Unix(1_700_000_000, 0)}, 1, 3)
	node := workerapp.NewNode("node", 1)
	a := &App{Jobs: q, Node: node}
	a.registerJobs()

	_, _ = q.Enqueue(context.Background(), jobdomain.Job{ID: "analysis", Kind: "analysis"})
	waitForRunning(t, node)
	_, _ = q.Cancel(context.Background(), "analysis")
	deadline := time.Now().Add(200 * time.Millisecond)
	for node.Snapshot().Running != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("node slot not released after cancellation, running=%d", node.Snapshot().Running)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitForRunning(t *testing.T, node *workerapp.Node) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for node.Snapshot().Running == 0 {
		if time.Now().After(deadline) {
			t.Fatal("node never started running")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
