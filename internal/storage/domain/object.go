package domain

import "time"

type Object struct {
	Key       string    `json:"key"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}
type Range struct{ Start, End int64 }

func (r Range) Valid(size int64) bool { return r.Start >= 0 && r.End >= r.Start && r.End <= size }
