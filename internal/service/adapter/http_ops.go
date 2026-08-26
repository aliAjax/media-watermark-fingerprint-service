package adapter

import (
	fingerprintdomain "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	ingestdomain "github.com/acme/media-watermark-fingerprinting/internal/ingest/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/httpx"
	app "github.com/acme/media-watermark-fingerprinting/internal/service/application"
	"io"
	"net/http"
	"strconv"
)

func (h *HTTP) createJob(w http.ResponseWriter, r *http.Request) error {
	var body app.JobRequest
	if err := decode(r, &body); err != nil {
		return err
	}
	j, err := h.app.CreateJob(r.Context(), body)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 202, j)
	return nil
}
func (h *HTTP) getJob(w http.ResponseWriter, r *http.Request, id string) error {
	j, err := h.app.GetJob(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, j)
	return nil
}
func (h *HTTP) cancelJob(w http.ResponseWriter, r *http.Request, id string) error {
	j, err := h.app.CancelJob(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, j)
	return nil
}
func (h *HTTP) startUpload(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	if err := decode(r, &body); err != nil {
		return err
	}
	u, err := h.app.Uploads.Start(r.Context(), body.ID, body.Name, body.Size)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 201, u)
	return nil
}
func (h *HTTP) appendChunk(w http.ResponseWriter, r *http.Request, id string) error {
	offset, err := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	u, err := h.app.Uploads.Append(r.Context(), id, ingestdomain.Chunk{Offset: offset, Data: data, Final: r.Header.Get("Upload-Final") == "true"})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, u)
	return nil
}
func (h *HTTP) publish(w http.ResponseWriter, r *http.Request, id string) error {
	cfg, err := h.app.Algorithms.Publish(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 200, cfg)
	return nil
}
func (h *HTTP) createAlgorithm(w http.ResponseWriter, r *http.Request) error {
	var body fingerprintdomain.AlgorithmConfig
	if err := decode(r, &body); err != nil {
		return err
	}
	if body.ID == "" {
		body.ID = "perceptual-v1"
	}
	if body.AudioWindow == 0 {
		body.AudioWindow = 500000000
	}
	if body.VideoWindow == 0 {
		body.VideoWindow = 8
	}
	if body.Threshold == 0 {
		body.Threshold = .78
	}
	cfg, err := h.app.Algorithms.CreateDraft(r.Context(), body)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, cfg)
	return nil
}
func (h *HTTP) rotateKey(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Secret string `json:"secret"`
	}
	if err := decode(r, &body); err != nil {
		return err
	}
	key, err := h.app.Keys.Rotate(r.Context(), []byte(body.Secret))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, 201, key)
	return nil
}

var _ = fingerprintdomain.DefaultConfig
