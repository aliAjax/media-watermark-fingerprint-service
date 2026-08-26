package application

import (
	"bytes"
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
)

type Parser interface {
	Parse(context.Context, []byte, domain.Limits) (domain.Metadata, error)
}
type Registry struct{ parsers map[domain.Format]Parser }

func NewRegistry() *Registry                                { return &Registry{parsers: make(map[domain.Format]Parser)} }
func (r *Registry) Register(format domain.Format, p Parser) { r.parsers[format] = p }
func (r *Registry) Detect(data []byte) domain.Format {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return domain.FormatWAV
	}
	if len(data) >= 8 && string(data[4:8]) == "ftyp" {
		return domain.FormatMP4
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return domain.FormatWebM
	}
	if len(data) >= 188 && data[0] == 0x47 && data[188%len(data)] == 0x47 {
		return domain.FormatMPEGTS
	}
	if len(data) >= 3 && (string(data[:3]) == "ID3" || (data[0] == 0xff && data[1]&0xe0 == 0xe0)) {
		return domain.FormatMP3
	}
	return domain.FormatUnknown
}
func (r *Registry) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	if int64(len(data)) > limits.MaxBytes {
		return domain.Metadata{}, fmt.Errorf("container limit: %w", domain.Limit("container exceeds maximum bytes"))
	}
	format := r.Detect(data)
	p, ok := r.parsers[format]
	if !ok {
		return domain.Metadata{}, fmt.Errorf("unsupported container: %w", domain.Unsupported("signature not recognized"))
	}
	m, err := p.Parse(ctx, data, limits)
	if err != nil {
		return domain.Metadata{}, fmt.Errorf("parse %s: %w", format, err)
	}
	m.Format = format
	m.Size = int64(len(data))
	if len(m.Tracks) > limits.MaxTracks {
		return domain.Metadata{}, fmt.Errorf("track limit: %w", domain.Limit("track count exceeds limit"))
	}
	if m.Duration > limits.MaxDuration {
		return domain.Metadata{}, fmt.Errorf("duration limit: %w", domain.Limit("duration exceeds limit"))
	}
	return m, nil
}
