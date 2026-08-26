package adapter

import (
	"context"
	"crypto/sha256"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/video/domain"
	"time"
)

type MockDecoder struct{}

func (MockDecoder) Decode(ctx context.Context, data []byte, meta containerdomain.Metadata, max int) ([]domain.Frame, error) {
	track := meta.VideoTrack()
	width, height := 32, 18
	if track != nil && track.Width > 0 && track.Height > 0 {
		if track.Width < 64 {
			width = track.Width
		}
		height = width * track.Height / track.Width
		if height < 8 {
			height = 8
		}
	}
	count := len(data)/256 + 1
	if count > max {
		count = max
	}
	if count > 120 {
		count = 120
	}
	seed := sha256.Sum256(data)
	frames := make([]domain.Frame, 0, count)
	for n := 0; n < count; n++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		luma := make([]byte, width*height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				p := (x*3 + y*5 + n*7 + int(seed[(x+y+n)%len(seed)])) & 255
				luma[y*width+x] = byte(p)
			}
		}
		frames = append(frames, domain.Frame{Index: n, PTS: time.Duration(n) * 40 * time.Millisecond, Width: width, Height: height, Luma: luma, Keyframe: n%25 == 0})
	}
	return frames, nil
}
