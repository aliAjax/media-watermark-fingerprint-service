package domain

import "time"

type State string

const (
	StateActive   State = "active"
	StateDraining State = "draining"
	StateStopped  State = "stopped"
)

type Node struct {
	ID        string    `json:"id"`
	State     State     `json:"state"`
	Running   int       `json:"running"`
	Capacity  int       `json:"capacity"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
