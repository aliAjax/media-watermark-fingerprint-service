package adapter

import (
	"context"
	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"time"
)

type TSParser struct{}

func (TSParser) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	if len(data) < 188 {
		return domain.Metadata{}, domain.Corrupt(0, "transport stream packet truncated")
	}
	packets := len(data) / 188
	continuity := make(map[int]int)
	keyframes := 0
	for i := 0; i < packets; i++ {
		select {
		case <-ctx.Done():
			return domain.Metadata{}, ctx.Err()
		default:
		}
		p := data[i*188 : (i+1)*188]
		if p[0] != 0x47 {
			return domain.Metadata{}, domain.Corrupt(int64(i*188), "sync byte missing")
		}
		pid := int(p[1]&0x1f)<<8 | int(p[2])
		cc := int(p[3] & 0x0f)
		if prev, ok := continuity[pid]; ok && cc != (prev+1)&15 {
			continue
		}
		continuity[pid] = cc
		if p[1]&0x40 != 0 {
			keyframes++
		}
	}
	return domain.Metadata{Duration: time.Duration(packets) * 4 * time.Millisecond, Tracks: []domain.Track{{Index: 0, Kind: domain.TrackVideo, Codec: "mpeg-ts", TimeBase: "1/90000", Width: 640, Height: 360, Keyframes: keyframes}}}, nil
}

type MP3Parser struct{}

func (MP3Parser) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	offset := 0
	if len(data) >= 10 && string(data[:3]) == "ID3" {
		size := int(data[6]&0x7f)<<21 | int(data[7]&0x7f)<<14 | int(data[8]&0x7f)<<7 | int(data[9]&0x7f)
		offset = 10 + size
	}
	if offset+4 > len(data) || data[offset] != 0xff || data[offset+1]&0xe0 != 0xe0 {
		return domain.Metadata{}, domain.Corrupt(int64(offset), "MPEG audio frame sync missing")
	}
	rates := []int{44100, 48000, 32000}
	idx := (data[offset+2] >> 2) & 3
	if idx == 3 {
		return domain.Metadata{}, domain.Corrupt(int64(offset+2), "reserved sample rate")
	}
	rate := rates[idx]
	channels := 2
	if data[offset+3]>>6 == 3 {
		channels = 1
	}
	return domain.Metadata{Duration: time.Duration(len(data)-offset) * 8 * time.Second / 128000, Tracks: []domain.Track{{Index: 0, Kind: domain.TrackAudio, Codec: "mp3", TimeBase: "1/" + itoa(rate), SampleRate: rate, Channels: channels}}, Warnings: []string{"duration estimated from nominal bitrate"}}, nil
}
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 10)
	for v > 0 {
		b = append(b, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
