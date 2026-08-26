package domain

import (
	"fmt"
	"time"
)

type FailureCode string

const (
	FailureAbsent      FailureCode = "absent"
	FailureCorrupt     FailureCode = "corrupt"
	FailureUnsupported FailureCode = "unsupported"
)

type Status string

const (
	StatusEmbedded    Status = "embedded"
	StatusDetected    Status = "detected"
	StatusAbsent      Status = "absent"
	StatusCorrupt     Status = "corrupt"
	StatusUnsupported Status = "unsupported"
)

type Payload struct {
	ClaimID    string `json:"claim_id"`
	Value      string `json:"value"`
	Nonce      string `json:"nonce"`
	Signature  string `json:"signature"`
	Algorithm  string `json:"algorithm"`
	KeyVersion int    `json:"key_version"`
}
type Result struct {
	Status     Status   `json:"status"`
	Payload    *Payload `json:"payload,omitempty"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason,omitempty"`
}
type Key struct {
	Version   int       `json:"version"`
	Secret    []byte    `json:"-"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

func (p Payload) Validate() error {
	if p.ClaimID == "" || p.Value == "" || p.Nonce == "" {
		return fmt.Errorf("claim_id, value and nonce are required")
	}
	if len(p.Value) > 256 {
		return fmt.Errorf("watermark value exceeds capacity")
	}
	return nil
}
