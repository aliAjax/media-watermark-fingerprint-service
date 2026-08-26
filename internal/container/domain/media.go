package domain

import "time"

type Format string

const (
	FormatUnknown Format = "unknown"
	FormatWAV     Format = "wav"
	FormatMP3     Format = "mp3"
	FormatMP4     Format = "mp4"
	FormatWebM    Format = "webm"
	FormatMPEGTS  Format = "mpeg-ts"
)

type TrackKind string

const (
	TrackAudio TrackKind = "audio"
	TrackVideo TrackKind = "video"
)

type Track struct {
	Index      int       `json:"index"`
	Kind       TrackKind `json:"kind"`
	Codec      string    `json:"codec"`
	TimeBase   string    `json:"time_base"`
	SampleRate int       `json:"sample_rate,omitempty"`
	Channels   int       `json:"channels,omitempty"`
	Width      int       `json:"width,omitempty"`
	Height     int       `json:"height,omitempty"`
	Keyframes  int       `json:"keyframes,omitempty"`
}
type Metadata struct {
	Format   Format        `json:"format"`
	Duration time.Duration `json:"duration"`
	Size     int64         `json:"size"`
	Tracks   []Track       `json:"tracks"`
	Warnings []string      `json:"warnings,omitempty"`
}
type Limits struct {
	MaxBytes    int64
	MaxDuration time.Duration
	MaxTracks   int
	MaxSamples  int
}

func DefaultLimits(maxBytes int64) Limits {
	return Limits{MaxBytes: maxBytes, MaxDuration: 4 * time.Hour, MaxTracks: 16, MaxSamples: 48_000 * 60 * 10}
}
func (m Metadata) AudioTrack() *Track {
	for i := range m.Tracks {
		if m.Tracks[i].Kind == TrackAudio {
			return &m.Tracks[i]
		}
	}
	return nil
}
func (m Metadata) VideoTrack() *Track {
	for i := range m.Tracks {
		if m.Tracks[i].Kind == TrackVideo {
			return &m.Tracks[i]
		}
	}
	return nil
}
