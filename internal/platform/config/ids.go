package config

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
	"time"
)

type IDs struct {
	prefix  string
	counter atomic.Uint64
}

func NewIDs(prefix string) *IDs { return &IDs{prefix: prefix} }
func (i *IDs) New() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err == nil {
		return i.prefix + "_" + hex.EncodeToString(b)
	}
	n := i.counter.Add(1)
	return i.prefix + "_" + time.Now().UTC().Format("20060102150405") + uintString(n)
}
func uintString(v uint64) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for v > 0 {
		b = append(b, byte('0'+v%10))
		v /= 10
	}
	for l, r := 0, len(b)-1; l < r; l, r = l+1, r-1 {
		b[l], b[r] = b[r], b[l]
	}
	return string(b)
}
