package adapter

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/watermark/application"
	"github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
)

type LSB struct{}

var magic = []byte("MWF1")

func (LSB) Embed(ctx context.Context, data []byte, p domain.Payload, key domain.Key) ([]byte, error) {
	payload := []byte(p.ClaimID + "\x00" + p.Value + "\x00" + p.Nonce + "\x00" + p.Signature + "\x00" + p.Algorithm)
	need := len(magic) + 2 + len(payload)
	if len(data) < need*8 {
		return nil, fmt.Errorf("carrier capacity %d bits below %d", len(data), need*8)
	}
	out := data
	header := make([]byte, 0, need)
	header = append(header, magic...)
	var size [2]byte
	binary.BigEndian.PutUint16(size[:], uint16(len(payload)))
	header = append(header, size[:]...)
	header = append(header, payload...)
	pos := 0
	for _, b := range header {
		for bit := 7; bit >= 0; bit-- {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			out[pos] = (out[pos] & 0xfe) | ((b >> uint(bit)) & 1)
			pos++
		}
	}
	return out, nil
}
func (LSB) Detect(ctx context.Context, data []byte, keys []domain.Key) (domain.Result, error) {
	if len(data) < (len(magic)+2)*8 {
		return domain.Result{Status: domain.StatusAbsent, Reason: "carrier too short"}, nil
	}
	raw := make([]byte, len(data)/8)
	for i := range raw {
		var b byte
		for bit := 0; bit < 8; bit++ {
			select {
			case <-ctx.Done():
				return domain.Result{}, ctx.Err()
			default:
			}
			b = (b << 1) | (data[i*8+bit] & 1)
		}
		raw[i] = b
	}
	if string(raw[:4]) != string(magic) {
		return domain.Result{Status: domain.StatusAbsent, Reason: "watermark magic not found"}, nil
	}
	size := int(binary.BigEndian.Uint16(raw[4:6]))
	if size <= 0 || 6+size > len(raw) {
		return domain.Result{Status: domain.StatusCorrupt, Reason: "declared payload outside carrier"}, nil
	}
	parts := split(string(raw[6 : 6+size]))
	if len(parts) != 5 {
		return domain.Result{Status: domain.StatusCorrupt, Reason: "watermark fields malformed"}, nil
	}
	p := domain.Payload{ClaimID: parts[0], Value: parts[1], Nonce: parts[2], Signature: parts[3], Algorithm: parts[4]}
	for _, k := range keys {
		if k.Version == 0 {
			continue
		}
		p.KeyVersion = k.Version
		if application.Verify(p, k.Secret) {
			return domain.Result{Status: domain.StatusDetected, Payload: &p, Confidence: 1}, nil
		}
	}
	return domain.Result{Status: domain.StatusCorrupt, Payload: &p, Confidence: .2, Reason: "payload signature invalid or key retired"}, nil
}
func split(s string) []string {
	out := make([]string, 0, 5)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == 0 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}
