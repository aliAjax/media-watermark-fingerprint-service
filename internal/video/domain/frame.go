package domain

import "time"

type Frame struct {
	Index    int           `json:"index"`
	PTS      time.Duration `json:"pts"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Luma     []byte        `json:"-"`
	Keyframe bool          `json:"keyframe"`
}

func (f Frame) AverageLuma() uint8 {
	if len(f.Luma) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range f.Luma {
		sum += uint64(v)
	}
	return uint8(sum / uint64(len(f.Luma)))
}
func (f Frame) Variance() float64 {
	if len(f.Luma) == 0 {
		return 0
	}
	mean := float64(f.AverageLuma())
	var total float64
	for _, v := range f.Luma {
		d := float64(v) - mean
		total += d * d
	}
	return total / float64(len(f.Luma))
}
func (f Frame) Rotate90() Frame {
	out := f
	out.Width, out.Height = f.Height, f.Width
	out.Luma = make([]byte, len(f.Luma))
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			src := y*f.Width + x
			dst := x*f.Height + (f.Height - 1 - y)
			if src < len(f.Luma) && dst < len(out.Luma) {
				out.Luma[dst] = f.Luma[src]
			}
		}
	}
	return out
}
