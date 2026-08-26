package application

import (
	"bytes"
	"context"
	"fmt"
	evidencedomain "github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
)

type EmbedRequest struct {
	ClaimID string
	Value   string
	Nonce   string
}
type EmbedResponse struct {
	AssetID       string         `json:"asset_id"`
	SourceAssetID string         `json:"source_asset_id"`
	Payload       domain.Payload `json:"payload"`
}

func (a *App) EmbedWatermark(ctx context.Context, assetID string, r EmbedRequest) (EmbedResponse, error) {
	source, err := a.GetAsset(ctx, assetID)
	if err != nil {
		return EmbedResponse{}, err
	}
	if source.Status != "active" {
		return EmbedResponse{}, fmt.Errorf("asset is %s", source.Status)
	}
	data, _, err := a.Objects.Bytes(ctx, source.ObjectKey)
	if err != nil {
		return EmbedResponse{}, err
	}
	out, payload, err := a.Watermarks.Embed(ctx, data, domain.Payload{ClaimID: r.ClaimID, Value: r.Value, Nonce: r.Nonce})
	if err != nil {
		return EmbedResponse{}, err
	}
	id := a.assetIDs.New()
	key := "assets/" + id + "/watermarked"
	obj, err := a.Objects.Put(ctx, key, bytes.NewReader(out), int64(len(out)))
	if err != nil {
		return EmbedResponse{}, err
	}
	derived := evidencedomain.Asset{ID: id, Name: source.Name + " (watermarked)", ObjectKey: key, SHA256: obj.SHA256, Size: obj.Size, Format: source.Format, Metadata: source.Metadata}
	created, err := a.Evidence.CreateAsset(ctx, derived)
	if err != nil {
		return EmbedResponse{}, err
	}
	return EmbedResponse{AssetID: created.ID, SourceAssetID: source.ID, Payload: payload}, nil
}
