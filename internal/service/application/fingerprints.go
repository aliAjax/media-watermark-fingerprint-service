package application

import (
	"context"
	"fmt"
	fingerprintapp "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/application"
	fingerprintdomain "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
)

func (a *App) CreateFingerprint(ctx context.Context, assetID string, kind fingerprintdomain.Kind) (fingerprintdomain.Fingerprint, error) {
	asset, err := a.GetAsset(ctx, assetID)
	if err != nil {
		return fingerprintdomain.Fingerprint{}, err
	}
	if asset.Status != "active" {
		return fingerprintdomain.Fingerprint{}, fmt.Errorf("asset is %s", asset.Status)
	}
	data, _, err := a.Objects.Bytes(ctx, asset.ObjectKey)
	if err != nil {
		return fingerprintdomain.Fingerprint{}, err
	}
	meta, err := a.Containers.Parse(ctx, data, a.limits)
	if err != nil {
		return fingerprintdomain.Fingerprint{}, err
	}
	cfg, err := a.Algorithms.Current(ctx, "perceptual-v1")
	if err != nil {
		return fingerprintdomain.Fingerprint{}, err
	}
	id := a.fpIDs.New()
	var fp fingerprintdomain.Fingerprint
	switch kind {
	case fingerprintdomain.KindAudio:
		samples, e := a.Audio.Decode(ctx, data, meta)
		if e != nil {
			return fp, e
		}
		fp = (fingerprintapp.AudioAlgorithm{}).Build(ctx, id, assetID, samples, cfg)
	case fingerprintdomain.KindVideo:
		frames, e := a.Video.Decode(ctx, data, meta)
		if e != nil {
			return fp, e
		}
		fp = (fingerprintapp.VideoAlgorithm{}).Build(ctx, id, assetID, frames, cfg)
	default:
		return fp, fmt.Errorf("fingerprint kind must be audio or video")
	}
	if len(fp.Segments) == 0 {
		return fp, fmt.Errorf("fingerprint produced no segments")
	}
	if err := a.Fingerprints.Save(ctx, fp); err != nil {
		return fp, err
	}
	return fp, nil
}
func (a *App) Match(ctx context.Context, fingerprintID string, threshold float64) ([]fingerprintdomain.Candidate, error) {
	fp, err := a.Fingerprints.Get(ctx, fingerprintID)
	if err != nil {
		return nil, err
	}
	return a.Matcher.Match(ctx, fingerprintdomain.MatchRequest{Query: fp, Threshold: threshold, Limit: 20})
}
