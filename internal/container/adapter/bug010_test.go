package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
)

func TestMP4ParserPreservesParseErrorIdentity(t *testing.T) {
	_, err := (MP4Parser{}).Parse(context.Background(), []byte("short"), domain.DefaultLimits(1024))
	var pe *domain.ParseError
	if !errors.As(err, &pe) || pe.Code != domain.CodeCorrupt {
		t.Fatalf("errors.As failed: %v", err)
	}
}
