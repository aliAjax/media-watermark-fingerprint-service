package infrastructure

import (
	"context"
	"sync"
	"testing"
)

func TestKeysCurrentDetachedDuringRotate(t *testing.T) {
	k := NewKeys([]byte("12345678"))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			_, _ = k.Current(context.Background())
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			_, _ = k.Rotate(context.Background(), []byte("abcdefgh"))
		}
	}()
	close(start)
	wg.Wait()
}

func TestKeysAllDetachedDuringRotate(t *testing.T) {
	k := NewKeys([]byte("12345678"))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			_, _ = k.All(context.Background())
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			_, _ = k.Rotate(context.Background(), []byte("abcdefgh"))
		}
	}()
	close(start)
	wg.Wait()
}
