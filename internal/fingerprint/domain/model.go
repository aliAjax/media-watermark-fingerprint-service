package domain

import "time"

type Kind string

const (
	KindAudio Kind = "audio"
	KindVideo Kind = "video"
)

type Segment struct {
	Start  time.Duration `json:"start"`
	End    time.Duration `json:"end"`
	Bits   uint64        `json:"bits"`
	Energy float64       `json:"energy"`
}
type Fingerprint struct {
	ID        string        `json:"id"`
	AssetID   string        `json:"asset_id"`
	Kind      Kind          `json:"kind"`
	Algorithm string        `json:"algorithm"`
	Version   string        `json:"version"`
	Window    time.Duration `json:"window"`
	Overlap   float64       `json:"overlap"`
	Segments  []Segment     `json:"segments"`
	CreatedAt time.Time     `json:"created_at"`
}
type Candidate struct {
	FingerprintID    string        `json:"fingerprint_id"`
	AssetID          string        `json:"asset_id"`
	QueryStart       time.Duration `json:"query_start"`
	CandidateStart   time.Duration `json:"candidate_start"`
	Duration         time.Duration `json:"duration"`
	Similarity       float64       `json:"similarity"`
	Threshold        float64       `json:"threshold"`
	AlgorithmVersion string        `json:"algorithm_version"`
	Matched          bool          `json:"matched"`
}
type MatchRequest struct {
	Query     Fingerprint
	Threshold float64
	Limit     int
}

func HammingSimilarity(a, b uint64) float64 {
	x := a ^ b
	count := 0
	for x != 0 {
		x &= x - 1
		count++
	}
	return 1 - float64(count)/64
}
func ClampSimilarity(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
