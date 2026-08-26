package adapter

import (
	"context"
	"sync"
	"testing"

	keyinfra "github.com/acme/media-watermark-fingerprinting/internal/watermark/infrastructure"
)

func TestDetectConcurrentWithRotate(t *testing.T) {
	keys := keyinfra.NewKeys([]byte("12345678"))
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			all, _ := keys.All(context.Background())
			_, _ = (LSB{}).Detect(context.Background(), data, all)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			_, _ = keys.Rotate(context.Background(), []byte("abcdefgh"))
		}
	}()
	close(start)
	wg.Wait()
}
