package adapter

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"time"
)

type WAVParser struct{}

func (WAVParser) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	if len(data) < 44 {
		return domain.Metadata{}, domain.Corrupt(int64(len(data)), "WAV header is truncated")
	}
	if string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return domain.Metadata{}, domain.Corrupt(0, "invalid RIFF/WAVE signature")
	}
	declared := int(binary.LittleEndian.Uint32(data[4:8])) + 8
	if declared > len(data) {
		return domain.Metadata{}, domain.Corrupt(4, "declared RIFF size exceeds input")
	}
	offset := 12
	var channels, sampleRate, bits int
	var dataSize int
	foundFmt, foundData := false, false
	for offset+8 <= len(data) {
		select {
		case <-ctx.Done():
			return domain.Metadata{}, ctx.Err()
		default:
		}
		id := string(data[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		body := offset + 8
		if size < 0 || body+size > len(data) {
			return domain.Metadata{}, domain.Corrupt(int64(offset+4), "chunk exceeds input")
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return domain.Metadata{}, domain.Corrupt(int64(body), "fmt chunk too short")
			}
			codec := binary.LittleEndian.Uint16(data[body : body+2])
			if codec != 1 {
				return domain.Metadata{}, domain.Unsupported(fmt.Sprintf("WAV codec %d", codec))
			}
			channels = int(binary.LittleEndian.Uint16(data[body+2 : body+4]))
			sampleRate = int(binary.LittleEndian.Uint32(data[body+4 : body+8]))
			bits = int(binary.LittleEndian.Uint16(data[body+14 : body+16]))
			foundFmt = true
		case "data":
			dataSize = size
			foundData = true
		}
		offset = body + size
		if offset%2 == 1 {
			offset++
		}
	}
	if !foundFmt || !foundData {
		return domain.Metadata{}, domain.Corrupt(12, "missing fmt or data chunk")
	}
	if channels < 1 || channels > 8 || sampleRate < 8000 || sampleRate > 192000 || bits < 8 || bits > 32 {
		return domain.Metadata{}, domain.Corrupt(20, "invalid audio parameters")
	}
	bytesPerSecond := sampleRate * channels * bits / 8
	if bytesPerSecond == 0 {
		return domain.Metadata{}, domain.Corrupt(28, "zero byte rate")
	}
	duration := time.Duration(float64(dataSize) / float64(bytesPerSecond) * float64(time.Second))
	return domain.Metadata{Duration: duration, Tracks: []domain.Track{{Index: 0, Kind: domain.TrackAudio, Codec: fmt.Sprintf("pcm_s%dle", bits), TimeBase: fmt.Sprintf("1/%d", sampleRate), SampleRate: sampleRate, Channels: channels}}}, nil
}
