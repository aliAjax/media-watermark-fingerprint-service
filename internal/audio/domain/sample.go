package domain

type Samples struct {
	Rate     int       `json:"rate"`
	Channels int       `json:"channels"`
	Values   []float32 `json:"-"`
}

func (s Samples) Frames() int {
	if s.Channels <= 0 {
		return 0
	}
	return len(s.Values) / s.Channels
}
func (s Samples) DurationSeconds() float64 {
	if s.Rate <= 0 {
		return 0
	}
	return float64(s.Frames()) / float64(s.Rate)
}
func (s Samples) Mono() []float32 {
	if s.Channels <= 1 {
		out := make([]float32, len(s.Values))
		copy(out, s.Values)
		return out
	}
	out := make([]float32, s.Frames())
	for i := range out {
		var sum float32
		for c := 0; c < s.Channels; c++ {
			sum += s.Values[i*s.Channels+c]
		}
		out[i] = sum / float32(s.Channels)
	}
	return out
}
func (s Samples) Peak() float32 {
	var p float32
	for _, v := range s.Values {
		if v < 0 {
			v = -v
		}
		if v > p {
			p = v
		}
	}
	return p
}
func (s *Samples) Normalize() {
	p := s.Peak()
	if p == 0 {
		return
	}
	scale := float32(.95) / p
	for i := range s.Values {
		s.Values[i] *= scale
	}
}
