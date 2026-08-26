package domain

import (
	"fmt"
	"time"
)

type AlgorithmConfig struct {
	ID          string        `json:"id"`
	Version     int           `json:"version"`
	State       string        `json:"state"`
	AudioWindow time.Duration `json:"audio_window"`
	VideoWindow int           `json:"video_window"`
	Overlap     float64       `json:"overlap"`
	Threshold   float64       `json:"threshold"`
	CreatedAt   time.Time     `json:"created_at"`
	PublishedAt *time.Time    `json:"published_at,omitempty"`
}

func (c AlgorithmConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("algorithm id is required")
	}
	if c.AudioWindow < 100*time.Millisecond || c.AudioWindow > 30*time.Second {
		return fmt.Errorf("audio window outside bounds")
	}
	if c.VideoWindow < 1 || c.VideoWindow > 300 {
		return fmt.Errorf("video window outside bounds")
	}
	if c.Overlap < 0 || c.Overlap >= .95 {
		return fmt.Errorf("overlap outside bounds")
	}
	if c.Threshold < .5 || c.Threshold > 1 {
		return fmt.Errorf("threshold outside bounds")
	}
	return nil
}
func DefaultConfig() AlgorithmConfig {
	return AlgorithmConfig{ID: "perceptual-v1", Version: 1, State: "published", AudioWindow: 500 * time.Millisecond, VideoWindow: 8, Overlap: .5, Threshold: .78, CreatedAt: time.Now().UTC()}
}
