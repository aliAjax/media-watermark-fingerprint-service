package application

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/audio/domain"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
)

type Decoder interface {
	Decode(context.Context, []byte, containerdomain.Metadata) (domain.Samples, error)
}
type Service struct {
	decoder    Decoder
	maxSamples int
}

func New(decoder Decoder, maxSamples int) *Service {
	return &Service{decoder: decoder, maxSamples: maxSamples}
}
func (s *Service) Decode(ctx context.Context, data []byte, meta containerdomain.Metadata) (domain.Samples, error) {
	if meta.AudioTrack() == nil {
		return domain.Samples{}, fmt.Errorf("media has no audio track")
	}
	samples, err := s.decoder.Decode(ctx, data, meta)
	if err != nil {
		return domain.Samples{}, fmt.Errorf("decode audio: %w", err)
	}
	if samples.Rate < 8000 || samples.Rate > 192000 || samples.Channels < 1 || samples.Channels > 8 {
		return domain.Samples{}, fmt.Errorf("decoder returned invalid format")
	}
	if len(samples.Values) > s.maxSamples*samples.Channels {
		return domain.Samples{}, fmt.Errorf("decoded sample limit exceeded")
	}
	return samples, nil
}
func Downsample(input []float32, factor int) []float32 {
	if factor <= 1 {
		return append([]float32(nil), input...)
	}
	out := make([]float32, 0, (len(input)+factor-1)/factor)
	for i := 0; i < len(input); i += factor {
		end := i + factor
		if end > len(input) {
			end = len(input)
		}
		var sum float32
		for _, v := range input[i:end] {
			sum += v
		}
		out = append(out, sum/float32(end-i))
	}
	return out
}
