package application

import (
	"bytes"
	"context"
	"fmt"
	evidencedomain "github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
)

type CreateAssetRequest struct {
	Name           string
	Data           []byte
	IdempotencyKey string
}

func (a *App) CreateAsset(ctx context.Context, r CreateAssetRequest) (evidencedomain.Asset, error) {
	if r.Name == "" {
		return evidencedomain.Asset{}, fmt.Errorf("asset name required")
	}
	if len(r.Data) == 0 {
		return evidencedomain.Asset{}, fmt.Errorf("asset data required")
	}
	meta, err := a.Containers.Parse(ctx, r.Data, a.limits)
	if err != nil {
		return evidencedomain.Asset{}, fmt.Errorf("parse media container: %w", err)
	}
	id := a.assetIDs.New()
	if r.IdempotencyKey != "" {
		id = "asset_" + safeKey(r.IdempotencyKey)
	}
	key := "assets/" + id + "/original"
	obj, err := a.Objects.Put(ctx, key, bytes.NewReader(r.Data), int64(len(r.Data)))
	if err != nil {
		return evidencedomain.Asset{}, fmt.Errorf("store media: %w", err)
	}
	asset := evidencedomain.Asset{ID: id, Name: r.Name, ObjectKey: key, SHA256: obj.SHA256, Size: obj.Size, Format: string(meta.Format), Metadata: meta}
	created, err := a.Evidence.CreateAsset(ctx, asset)
	if err != nil {
		return evidencedomain.Asset{}, err
	}
	return created, nil
}
func safeKey(value string) string {
	b := make([]byte, 0, len(value))
	for i := 0; i < len(value) && len(b) < 48; i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			b = append(b, c)
		}
	}
	if len(b) == 0 {
		return "key"
	}
	return string(b)
}
func (a *App) FreezeAsset(ctx context.Context, id, actor string) (evidencedomain.Asset, error) {
	return a.Evidence.ChangeStatus(ctx, id, evidencedomain.AssetFrozen, actor)
}
func (a *App) WithdrawAsset(ctx context.Context, id, actor string) (evidencedomain.Asset, error) {
	return a.Evidence.ChangeStatus(ctx, id, evidencedomain.AssetWithdrawn, actor)
}
