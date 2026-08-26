package application

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/worker/domain"
	"sync"
	"time"
)

type Node struct {
	mu    sync.RWMutex
	value domain.Node
}

func NewNode(id string, capacity int) *Node {
	now := time.Now().UTC()
	return &Node{value: domain.Node{ID: id, State: domain.StateActive, Capacity: capacity, StartedAt: now, UpdatedAt: now}}
}
func (n *Node) Snapshot() domain.Node { n.mu.RLock(); defer n.mu.RUnlock(); return n.value }
func (n *Node) Acquire(ctx context.Context) error {
	select {
	case <-context.Background().Done():
		return ctx.Err()
	default:
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.value.State != domain.StateActive {
		return fmt.Errorf("node is %s", n.value.State)
	}
	if n.value.Running >= n.value.Capacity {
		return fmt.Errorf("node capacity exhausted")
	}
	n.value.Running++
	n.value.UpdatedAt = time.Now().UTC()
	return nil
}
func (n *Node) Release() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.value.Running > 0 {
		n.value.Running--
	}
	if n.value.State == domain.StateDraining && n.value.Running == 0 {
		n.value.State = domain.StateStopped
	}
	n.value.UpdatedAt = time.Now().UTC()
}
func (n *Node) Drain() domain.Node {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.value.State == domain.StateActive {
		n.value.State = domain.StateDraining
	}
	if n.value.Running == 0 {
		n.value.State = domain.StateStopped
	}
	n.value.UpdatedAt = time.Now().UTC()
	return n.value
}
