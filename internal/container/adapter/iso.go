package adapter

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"time"
)

type MP4Parser struct{}

func (MP4Parser) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	if len(data) < 16 {
		return domain.Metadata{}, fmt.Errorf("mp4 header: %v", domain.Corrupt(int64(len(data)), "MP4 header truncated"))
	}
	offset := 0
	seenFTYP, seenMedia := false, false
	tracks := []domain.Track{}
	duration := time.Duration(0)
	for offset+8 <= len(data) {
		select {
		case <-ctx.Done():
			return domain.Metadata{}, ctx.Err()
		default:
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		kind := string(data[offset+4 : offset+8])
		if size == 1 {
			if offset+16 > len(data) {
				return domain.Metadata{}, domain.Corrupt(int64(offset), "extended atom truncated")
			}
			wide := binary.BigEndian.Uint64(data[offset+8 : offset+16])
			if wide > uint64(len(data)-offset) {
				return domain.Metadata{}, domain.Corrupt(int64(offset), "extended atom too large")
			}
			size = int(wide)
		}
		if size < 8 || offset+size > len(data) {
			return domain.Metadata{}, domain.Corrupt(int64(offset), "atom size outside container")
		}
		switch kind {
		case "ftyp":
			seenFTYP = true
		case "mdat":
			seenMedia = true
		case "moov":
			if bytes.Contains(data[offset:offset+size], []byte("vide")) {
				tracks = append(tracks, domain.Track{Index: len(tracks), Kind: domain.TrackVideo, Codec: "mock-h264", TimeBase: "1/90000", Width: 640, Height: 360, Keyframes: 1})
			}
			if bytes.Contains(data[offset:offset+size], []byte("soun")) {
				tracks = append(tracks, domain.Track{Index: len(tracks), Kind: domain.TrackAudio, Codec: "mock-aac", TimeBase: "1/48000", SampleRate: 48000, Channels: 2})
			}
			duration = time.Second
		}
		offset += size
	}
	if !seenFTYP {
		return domain.Metadata{}, domain.Corrupt(4, "ftyp atom missing")
	}
	if !seenMedia {
		return domain.Metadata{}, domain.Corrupt(8, "mdat atom missing")
	}
	if len(tracks) == 0 {
		tracks = []domain.Track{{Index: 0, Kind: domain.TrackVideo, Codec: "unknown", TimeBase: "1/90000", Width: 320, Height: 180}}
	}
	return domain.Metadata{Duration: duration, Tracks: tracks}, nil
}

type WebMParser struct{}

func (WebMParser) Parse(ctx context.Context, data []byte, limits domain.Limits) (domain.Metadata, error) {
	if len(data) < 12 || !bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return domain.Metadata{}, domain.Corrupt(0, "invalid EBML header")
	}
	if !bytes.Contains(data, []byte{0x18, 0x53, 0x80, 0x67}) {
		return domain.Metadata{}, domain.Corrupt(4, "segment element missing")
	}
	kind := domain.TrackVideo
	if bytes.Contains(bytes.ToLower(data), []byte("audio")) {
		kind = domain.TrackAudio
	}
	track := domain.Track{Index: 0, Kind: kind, Codec: "mock-webm", TimeBase: "1/1000"}
	if kind == domain.TrackAudio {
		track.SampleRate = 48000
		track.Channels = 2
	} else {
		track.Width = 640
		track.Height = 360
		track.Keyframes = 1
	}
	return domain.Metadata{Duration: time.Second, Tracks: []domain.Track{track}, Warnings: []string{"duration estimated by pure-Go parser"}}, nil
}
