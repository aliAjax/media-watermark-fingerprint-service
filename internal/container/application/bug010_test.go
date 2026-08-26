package application

import (
	"context"
	"errors"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
)

func TestRegistryParsePreservesUnsupportedError(t *testing.T) {
	r := NewRegistry()
	_, err := r.Parse(context.Background(), []byte("unknown"), domain.DefaultLimits(1024))
	var pe *domain.ParseError
	if !errors.As(err, &pe) || pe.Code != domain.CodeUnsupported {
		t.Fatalf("errors.As failed: %v", err)
	}
}

func TestRegistryParsePreservesLimitError(t *testing.T) {
	r := NewRegistry()
	_, err := r.Parse(context.Background(), make([]byte, 2048), domain.DefaultLimits(1024))
	var pe *domain.ParseError
	if !errors.As(err, &pe) || pe.Code != domain.CodeLimit {
		t.Fatalf("errors.As failed: %v", err)
	}
}

type corruptParser struct{}

func (corruptParser) Parse(context.Context, []byte, domain.Limits) (domain.Metadata, error) {
	return domain.Metadata{}, domain.Corrupt(0, "fake corrupt")
}

func TestRegistryParsePreservesCorruptError(t *testing.T) {
	r := NewRegistry()
	r.Register(domain.FormatUnknown, corruptParser{})
	data := []byte("unknown")
	_, err := r.Parse(context.Background(), data, domain.DefaultLimits(1024))
	var pe *domain.ParseError
	if !errors.As(err, &pe) || pe.Code != domain.CodeCorrupt {
		t.Fatalf("errors.As failed: %v", err)
	}
}

type longParser struct{}

func (longParser) Parse(context.Context, []byte, domain.Limits) (domain.Metadata, error) {
	return domain.Metadata{Duration: 5 * 3600 * 1_000_000_000}, nil
}

func TestRegistryParsePreservesDurationLimitError(t *testing.T) {
	r := NewRegistry()
	r.Register(domain.FormatUnknown, longParser{})
	_, err := r.Parse(context.Background(), []byte("unknown"), domain.DefaultLimits(1024))
	var pe *domain.ParseError
	if !errors.As(err, &pe) || pe.Code != domain.CodeLimit {
		t.Fatalf("errors.As failed: %v", err)
	}
}
