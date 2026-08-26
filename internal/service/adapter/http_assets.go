package adapter

import (
	"fmt"
	fingerprintdomain "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/httpx"
	app "github.com/acme/media-watermark-fingerprinting/internal/service/application"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (h *HTTP) createAsset(w http.ResponseWriter, r *http.Request) error {
	name := r.Header.Get("X-Asset-Name")
	if name == "" {
		name = "upload.bin"
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read asset: %w", err)
	}
	asset, err := h.app.CreateAsset(r.Context(), app.CreateAssetRequest{Name: name, Data: data, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, asset)
	return nil
}
func (h *HTTP) assetRoutes(w http.ResponseWriter, r *http.Request, parts []string) error {
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		asset, err := h.app.GetAsset(r.Context(), id)
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 200, asset)
		return nil
	}
	if len(parts) == 2 && parts[1] == "fingerprints" && r.Method == http.MethodPost {
		var body struct {
			Kind fingerprintdomain.Kind `json:"kind"`
		}
		if err := decode(r, &body); err != nil {
			return err
		}
		fp, err := h.app.CreateFingerprint(r.Context(), id, body.Kind)
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 201, fp)
		return nil
	}
	if len(parts) == 3 && parts[1] == "watermarks" && parts[2] == "embed" && r.Method == http.MethodPost {
		var body app.EmbedRequest
		if err := decode(r, &body); err != nil {
			return err
		}
		result, err := h.app.EmbedWatermark(r.Context(), id, body)
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 201, result)
		return nil
	}
	if len(parts) == 2 && parts[1] == "freeze" && r.Method == http.MethodPost {
		asset, err := h.app.FreezeAsset(r.Context(), id, r.Header.Get("X-Actor"))
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 200, asset)
		return nil
	}
	if len(parts) == 2 && parts[1] == "withdraw" && r.Method == http.MethodPost {
		asset, err := h.app.WithdrawAsset(r.Context(), id, r.Header.Get("X-Actor"))
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 200, asset)
		return nil
	}
	if len(parts) == 2 && parts[1] == "timeline" && r.Method == http.MethodGet {
		events, err := h.app.Evidence.Timeline(r.Context(), id)
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, 200, map[string]any{"events": events})
		return nil
	}
	return &httpx.CodeError{Code: "route_not_found", Message: "asset route not found", Status: 404}
}
func (h *HTTP) detect(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		AssetID string `json:"asset_id"`
	}
	if err := decode(r, &body); err != nil {
		return err
	}
	result, err := h.app.DetectWatermark(r.Context(), body.AssetID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, result)
	return nil
}
func (h *HTTP) match(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		FingerprintID string  `json:"fingerprint_id"`
		Threshold     float64 `json:"threshold"`
	}
	if err := decode(r, &body); err != nil {
		return err
	}
	candidates, err := h.app.Match(r.Context(), body.FingerprintID, body.Threshold)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, map[string]any{"candidates": candidates, "count": len(candidates)})
	return nil
}

var _ = strconv.Itoa
var _ = strings.TrimSpace
