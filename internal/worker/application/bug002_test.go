package application

import (
	"context"
	"testing"
)

func TestNodeAcquireHonorsCancellation(t *testing.T) {
	n := NewNode("node", 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := n.Acquire(ctx); err == nil {
		t.Fatal("Acquire returned nil error for canceled context")
	}
	if got := n.Snapshot().Running; got != 0 {
		t.Fatalf("running=%d after canceled acquire", got)
	}
}
