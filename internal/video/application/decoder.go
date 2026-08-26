package application

import (
	"context"
	"fmt"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/video/domain"
)

type Decoder interface {
	Decode(context.Context, []byte, containerdomain.Metadata, int) ([]domain.Frame, error)
}
type Service struct {
	decoder   Decoder
	maxFrames int
}

func New(decoder Decoder, maxFrames int) *Service {
	return &Service{decoder: decoder, maxFrames: maxFrames}
}
func (s *Service) Decode(ctx context.Context, data []byte, meta containerdomain.Metadata) ([]domain.Frame, error) {
	if meta.VideoTrack() == nil {
		return nil, fmt.Errorf("media has no video track")
	}
	frames, err := s.decoder.Decode(ctx, data, meta, s.maxFrames)
	if err != nil {
		return nil, fmt.Errorf("decode video: %w", err)
	}
	if len(frames) > s.maxFrames {
		return nil, fmt.Errorf("decoded frame limit exceeded")
	}
	for i, f := range frames {
		if f.Width < 1 || f.Height < 1 || len(f.Luma) != f.Width*f.Height {
			return nil, fmt.Errorf("frame %d has invalid dimensions", i)
		}
	}
	return frames, nil
}
func Thumbnail(frame domain.Frame, width int) domain.Frame {
	if width <= 0 || width >= frame.Width {
		return frame
	}
	height := frame.Height * width / frame.Width
	if height < 1 {
		height = 1
	}
	out := domain.Frame{Index: frame.Index, PTS: frame.PTS, Width: width, Height: height, Keyframe: frame.Keyframe, Luma: make([]byte, width*height)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := x * frame.Width / width
			sy := y * frame.Height / height
			out.Luma[y*width+x] = frame.Luma[sy*frame.Width+sx]
		}
	}
	return out
}
