package application

import (
	"context"
	"testing"

	audiodomain "github.com/acme/media-watermark-fingerprinting/internal/audio/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
)

func TestAudioBuildUsesCurrentVersion(t *testing.T) {
	cfg := domain.DefaultConfig()
	cfg.Version = 2
	samples := audiodomain.Samples{Rate: 8000, Channels: 1, Values: make([]float32, 256)}
	fp := (AudioAlgorithm{}).Build(context.Background(), "fp1", "asset1", samples, cfg)
	if fp.Version != "a2" {
		t.Fatalf("version = %s, want a2", fp.Version)
	}
}
