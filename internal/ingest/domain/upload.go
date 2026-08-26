package domain

import "time"

type Status string

const (
	StatusOpen     Status = "open"
	StatusComplete Status = "complete"
	StatusAborted  Status = "aborted"
)

type Upload struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ExpectedSize int64     `json:"expected_size"`
	ReceivedSize int64     `json:"received_size"`
	SHA256       string    `json:"sha256,omitempty"`
	ObjectKey    string    `json:"object_key,omitempty"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type Chunk struct {
	Offset int64
	Data   []byte
	Final  bool
}
