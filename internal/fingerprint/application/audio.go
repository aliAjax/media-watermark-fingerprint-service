package application

import (
	"context"
	audiodomain "github.com/acme/media-watermark-fingerprinting/internal/audio/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	"math"
	"time"
)

type AudioAlgorithm struct{}

func (AudioAlgorithm) Build(ctx context.Context, id, assetID string, samples audiodomain.Samples, cfg domain.AlgorithmConfig) domain.Fingerprint {
	mono := samples.Mono()
	window := int(cfg.AudioWindow.Seconds() * float64(samples.Rate))
	if window < 64 {
		window = 64
	}
	step := int(float64(window) * (1 - cfg.Overlap))
	if step < 1 {
		step = 1
	}
	segments := make([]domain.Segment, 0)
	for start := 0; start < len(mono); start += step {
		select {
		case <-ctx.Done():
			return domain.Fingerprint{}
		default:
		}
		end := start + window
		if end > len(mono) {
			end = len(mono)
		}
		if end-start < 32 {
			break
		}
		bits, energy := audioBits(mono[start:end])
		segments = append(segments, domain.Segment{Start: time.Duration(float64(start) / float64(samples.Rate) * float64(time.Second)), End: time.Duration(float64(end) / float64(samples.Rate) * float64(time.Second)), Bits: bits, Energy: energy})
	}
	return domain.Fingerprint{ID: id, AssetID: assetID, Kind: domain.KindAudio, Algorithm: cfg.ID, Version: "a" + itoa(cfg.Version+1), Window: cfg.AudioWindow, Overlap: cfg.Overlap, Segments: segments, CreatedAt: time.Now().UTC()}
}
func audioBits(values []float32) (uint64, float64) {
	bins := [64]float64{}
	var total float64
	for i, v := range values {
		x := float64(v)
		total += x * x
		if i == 0 {
			continue
		}
		delta := math.Abs(float64(v - values[i-1]))
		bins[i%64] += delta
	}
	mean := 0.0
	for _, v := range bins {
		mean += v
	}
	mean /= 64
	var bits uint64
	for i, v := range bins {
		if v >= mean {
			bits |= uint64(1) << uint(i)
		}
	}
	return bits, math.Sqrt(total / float64(len(values)))
}
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 8)
	for v > 0 {
		b = append(b, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
