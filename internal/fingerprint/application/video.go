package application

import (
	"context"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	videodomain "github.com/acme/media-watermark-fingerprinting/internal/video/domain"
	"math"
	"time"
)

type VideoAlgorithm struct{}

func (VideoAlgorithm) Build(ctx context.Context, id, assetID string, frames []videodomain.Frame, cfg domain.AlgorithmConfig) domain.Fingerprint {
	step := int(float64(cfg.VideoWindow) * (1 - cfg.Overlap))
	if step < 1 {
		step = 1
	}
	segments := make([]domain.Segment, 0)
	for start := 0; start < len(frames); start += step {
		select {
		case <-ctx.Done():
			return domain.Fingerprint{}
		default:
		}
		end := start + cfg.VideoWindow
		if end > len(frames) {
			end = len(frames)
		}
		if end-start < 1 {
			break
		}
		bits, energy := videoBits(frames[start:end])
		segments = append(segments, domain.Segment{Start: frames[start].PTS, End: frames[end-1].PTS + 40*time.Millisecond, Bits: bits, Energy: energy})
	}
	return domain.Fingerprint{ID: id, AssetID: assetID, Kind: domain.KindVideo, Algorithm: cfg.ID, Version: "v" + itoa(cfg.Version), Window: time.Duration(cfg.VideoWindow) * 40 * time.Millisecond, Overlap: cfg.Overlap, Segments: segments, CreatedAt: time.Now().UTC()}
}
func videoBits(frames []videodomain.Frame) (uint64, float64) {
	grid := [64]float64{}
	variance := 0.0
	for _, frame := range frames {
		variance += frame.Variance()
		for gy := 0; gy < 8; gy++ {
			for gx := 0; gx < 8; gx++ {
				x := gx * frame.Width / 8
				y := gy * frame.Height / 8
				if x >= frame.Width {
					x = frame.Width - 1
				}
				if y >= frame.Height {
					y = frame.Height - 1
				}
				grid[gy*8+gx] += float64(frame.Luma[y*frame.Width+x])
			}
		}
	}
	mean := 0.0
	for _, v := range grid {
		mean += v
	}
	mean /= 64
	var bits uint64
	for i, v := range grid {
		if v >= mean {
			bits |= uint64(1) << uint(i)
		}
	}
	return canonicalRotations(bits), math.Sqrt(variance / float64(len(frames)))
}
func canonicalRotations(bits uint64) uint64 {
	best := bits
	current := bits
	for r := 0; r < 3; r++ {
		var next uint64
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				src := y*8 + x
				dst := x*8 + (7 - y)
				if current&(uint64(1)<<uint(src)) != 0 {
					next |= uint64(1) << uint(dst)
				}
			}
		}
		if next < best {
			best = next
		}
		current = next
	}
	return best
}
