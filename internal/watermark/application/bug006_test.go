package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
)

var errEmbed = errors.New("carrier capacity exceeded")

type fakeKeys struct{}

func (fakeKeys) Current(context.Context) (domain.Key, error) {
	return domain.Key{Version: 1, Secret: []byte("secret"), Active: true}, nil
}
func (fakeKeys) All(context.Context) ([]domain.Key, error) {
	return []domain.Key{{Version: 1, Secret: []byte("secret"), Active: true}}, nil
}
func (fakeKeys) Rotate(context.Context, []byte) (domain.Key, error) { return domain.Key{}, nil }

type fakeAlgo struct{ err error }

func (f fakeAlgo) Embed(context.Context, []byte, domain.Payload, domain.Key) ([]byte, error) {
	return nil, f.err
}
func (f fakeAlgo) Detect(context.Context, []byte, []domain.Key) (domain.Result, error) {
	return domain.Result{Status: domain.StatusDetected}, nil
}

func TestWatermarkEmbedErrorDoesNotOverwrite(t *testing.T) {
	s := New(fakeKeys{}, fakeAlgo{err: errEmbed}, RealClock{})
	_, _, err := s.Embed(context.Background(), make([]byte, 64), domain.Payload{ClaimID: "c", Value: "v", Nonce: "n"})
	if err == nil {
		t.Fatal("expected embed error")
	}
	if !errors.Is(err, errEmbed) {
		t.Fatalf("original embed error identity lost after cleanup: %v", err)
	}
	if !strings.Contains(err.Error(), errEmbed.Error()) {
		t.Fatalf("original embed error message hidden: %v", err)
	}
}
