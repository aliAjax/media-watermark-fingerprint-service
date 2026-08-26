package adapter

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/audio/domain"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	"math"
)

type MockDecoder struct{}

func (MockDecoder) Decode(ctx context.Context, data []byte, meta containerdomain.Metadata) (domain.Samples, error) {
	track := meta.AudioTrack()
	if track == nil {
		return domain.Samples{}, fmt.Errorf("audio track absent")
	}
	rate, channels := track.SampleRate, track.Channels
	if rate == 0 {
		rate = 16000
	}
	if channels == 0 {
		channels = 1
	}
	if meta.Format == containerdomain.FormatWAV {
		return decodePCM(ctx, data, rate, channels)
	}
	count := len(data) * 2
	if count > rate*channels*10 {
		count = rate * channels * 10
	}
	values := make([]float32, count)
	seed := uint32(2166136261)
	for i, b := range data {
		seed ^= uint32(b)
		seed *= 16777619
		if i%1024 == 0 {
			select {
			case <-ctx.Done():
				return domain.Samples{}, ctx.Err()
			default:
			}
		}
	}
	for i := range values {
		x := float64(i) / float64(rate*channels)
		carrier := math.Sin(2 * math.Pi * 440 * x)
		texture := float64(int(seed>>uint(i%24)&255)-128) / 2048
		values[i] = float32(carrier*.65 + texture)
	}
	return domain.Samples{Rate: rate, Channels: channels, Values: values}, nil
}
func decodePCM(ctx context.Context, data []byte, rate, channels int) (domain.Samples, error) {
	if len(data) < 44 {
		return domain.Samples{}, fmt.Errorf("WAV too short")
	}
	bits := int(binary.LittleEndian.Uint16(data[34:36]))
	offset := 12
	start, size := 0, 0
	for offset+8 <= len(data) {
		n := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if offset+8+n > len(data) {
			break
		}
		if string(data[offset:offset+4]) == "data" {
			start, size = offset+8, n
			break
		}
		offset += 8 + n
		if offset%2 == 1 {
			offset++
		}
	}
	if start == 0 {
		return domain.Samples{}, fmt.Errorf("data chunk absent")
	}
	step := bits / 8
	if step != 1 && step != 2 {
		return domain.Samples{}, fmt.Errorf("mock decoder supports 8/16 bit PCM")
	}
	values := make([]float32, 0, size/step)
	for i := start; i+step <= start+size; i += step {
		if len(values)%4096 == 0 {
			select {
			case <-ctx.Done():
				return domain.Samples{}, ctx.Err()
			default:
			}
		}
		if step == 1 {
			values = append(values, (float32(data[i])-128)/128)
		} else {
			v := int16(binary.LittleEndian.Uint16(data[i : i+2]))
			values = append(values, float32(v)/32768)
		}
	}
	return domain.Samples{Rate: rate, Channels: channels, Values: values}, nil
}
