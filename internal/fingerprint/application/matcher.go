package application

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	"sort"
	"time"
)

type Repository interface {
	Save(context.Context, domain.Fingerprint) error
	Get(context.Context, string) (domain.Fingerprint, error)
	List(context.Context) ([]domain.Fingerprint, error)
}
type Matcher struct{ repo Repository }

func NewMatcher(repo Repository) *Matcher { return &Matcher{repo: repo} }
func (m *Matcher) Match(ctx context.Context, request domain.MatchRequest) ([]domain.Candidate, error) {
	if len(request.Query.Segments) == 0 {
		return nil, fmt.Errorf("query fingerprint has no segments")
	}
	if request.Threshold == 0 {
		request.Threshold = .78
	}
	if request.Threshold < 0 || request.Threshold > 1 {
		return nil, fmt.Errorf("threshold outside 0..1")
	}
	if request.Limit <= 0 || request.Limit > 100 {
		request.Limit = 10
	}
	all, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list fingerprints: %w", err)
	}
	candidates := make([]domain.Candidate, 0)
	for _, target := range all {
		if target.ID == request.Query.ID || target.Kind != request.Query.Kind || target.Algorithm != request.Query.Algorithm {
			continue
		}
		candidate := compare(request.Query, target, request.Threshold)
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Similarity > candidates[j].Similarity })
	if len(candidates) > request.Limit {
		candidates = candidates[:request.Limit]
	}
	return candidates, nil
}
func compare(query, target domain.Fingerprint, threshold float64) domain.Candidate {
	best, qi, ti := 0.0, 0, 0
	for i, q := range query.Segments {
		for j, t := range target.Segments {
			sim := domain.HammingSimilarity(q.Bits, t.Bits)
			energyRatio := q.Energy / t.Energy
			if t.Energy == 0 {
				energyRatio = 1
			}
			if energyRatio > 1 {
				energyRatio = 1 / energyRatio
			}
			sim = domain.ClampSimilarity(sim*.9 + energyRatio*.1)
			if sim > best {
				best, qi, ti = sim, i, j
			}
		}
	}
	duration := timeMin(query.Segments[qi].End-query.Segments[qi].Start, target.Segments[ti].End-target.Segments[ti].Start)
	return domain.Candidate{FingerprintID: target.ID, AssetID: target.AssetID, QueryStart: query.Segments[qi].Start, CandidateStart: target.Segments[ti].Start, Duration: duration, Similarity: best, Threshold: threshold, AlgorithmVersion: target.Version, Matched: best >= threshold}
}
func timeMin(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
