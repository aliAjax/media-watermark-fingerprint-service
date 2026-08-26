package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
	"strings"
	"time"
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type KeyStore interface {
	Current(context.Context) (domain.Key, error)
	All(context.Context) ([]domain.Key, error)
	Rotate(context.Context, []byte) (domain.Key, error)
}
type Algorithm interface {
	Embed(context.Context, []byte, domain.Payload, domain.Key) ([]byte, error)
	Detect(context.Context, []byte, []domain.Key) (domain.Result, error)
}
type Service struct {
	keys      KeyStore
	algorithm Algorithm
	clock     Clock
}

func New(keys KeyStore, algorithm Algorithm, clock Clock) *Service {
	return &Service{keys: keys, algorithm: algorithm, clock: clock}
}
func (s *Service) Embed(ctx context.Context, data []byte, p domain.Payload) ([]byte, domain.Payload, error) {
	if err := p.Validate(); err != nil {
		return nil, p, err
	}
	key, err := s.keys.Current(ctx)
	if err != nil {
		return nil, p, fmt.Errorf("current watermark key: %w", err)
	}
	p.KeyVersion = key.Version
	p.Algorithm = "blind-lsb-v1"
	p.Signature = sign(p, key.Secret)
	out, err := s.algorithm.Embed(ctx, data, p, key)
	if err != nil {
		return nil, p, fmt.Errorf("embed watermark: %w", err)
	}
	return out, p, nil
}
func (s *Service) Detect(ctx context.Context, data []byte) (domain.Result, error) {
	keys, err := s.keys.All(ctx)
	if err != nil {
		return domain.Result{}, fmt.Errorf("watermark keys: %w", err)
	}
	r, err := s.algorithm.Detect(ctx, data, keys)
	if err != nil {
		return domain.Result{}, fmt.Errorf("detect watermark: %w", err)
	}
	return r, nil
}
func sign(p domain.Payload, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(strings.Join([]string{p.ClaimID, p.Value, p.Nonce}, "|")))
	return hex.EncodeToString(h.Sum(nil))
}
func Verify(p domain.Payload, secret []byte) bool {
	want := sign(p, secret)
	return subtle.ConstantTimeCompare([]byte(want), []byte(p.Signature)) == 1
}
