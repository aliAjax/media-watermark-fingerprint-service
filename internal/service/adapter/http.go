package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/httpx"
	app "github.com/acme/media-watermark-fingerprinting/internal/service/application"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

type HTTP struct {
	app      *app.App
	requests atomic.Uint64
	failures atomic.Uint64
}

func NewHTTP(a *app.App) *HTTP { return &HTTP{app: a} }
func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requests.Add(1)
	if err := h.route(w, r); err != nil {
		h.failures.Add(1)
		httpx.WriteError(w, r, normalize(err))
	}
}
func (h *HTTP) route(w http.ResponseWriter, r *http.Request) error {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if r.Method == http.MethodGet && path == "healthz" {
		httpx.WriteJSON(w, 200, map[string]any{"status": "ok", "uptime_seconds": int(h.app.Uptime().Seconds())})
		return nil
	}
	if r.Method == http.MethodGet && path == "readyz" {
		if !h.app.Ready() {
			return &httpx.CodeError{Code: "not_ready", Message: "service is not ready", Status: 503}
		}
		httpx.WriteJSON(w, 200, map[string]string{"status": "ready"})
		return nil
	}
	if r.Method == http.MethodGet && path == "metrics" {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "mwf_http_requests_total %d\nmwf_http_failures_total %d\n", h.requests.Load(), h.failures.Load())
		return nil
	}
	if r.Method == http.MethodPost && path == "v1/assets" {
		return h.createAsset(w, r)
	}
	if len(parts) >= 3 && parts[0] == "v1" && parts[1] == "assets" {
		return h.assetRoutes(w, r, parts[2:])
	}
	if r.Method == http.MethodPost && path == "v1/watermarks/detect" {
		return h.detect(w, r)
	}
	if r.Method == http.MethodPost && path == "v1/matches" {
		return h.match(w, r)
	}
	if r.Method == http.MethodPost && path == "v1/jobs" {
		return h.createJob(w, r)
	}
	if len(parts) == 3 && parts[0] == "v1" && parts[1] == "jobs" && r.Method == http.MethodGet {
		return h.getJob(w, r, parts[2])
	}
	if len(parts) == 4 && parts[0] == "v1" && parts[1] == "jobs" && parts[3] == "cancel" && r.Method == http.MethodPost {
		return h.cancelJob(w, r, parts[2])
	}
	if r.Method == http.MethodPost && path == "v1/uploads" {
		return h.startUpload(w, r)
	}
	if len(parts) == 4 && parts[0] == "v1" && parts[1] == "uploads" && parts[3] == "chunks" && r.Method == http.MethodPut {
		return h.appendChunk(w, r, parts[2])
	}
	if len(parts) == 4 && parts[0] == "v1" && parts[1] == "algorithms" && parts[3] == "publish" && r.Method == http.MethodPost {
		return h.publish(w, r, parts[2])
	}
	if r.Method == http.MethodPost && path == "v1/watermark-keys/rotate" {
		return h.rotateKey(w, r)
	}
	if r.Method == http.MethodPost && path == "v1/algorithms" {
		return h.createAlgorithm(w, r)
	}
	if r.Method == http.MethodPost && path == "v1/nodes/current/drain" {
		httpx.WriteJSON(w, 200, h.app.Node.Drain())
		return nil
	}
	return &httpx.CodeError{Code: "route_not_found", Message: "route not found", Status: 404}
}
func normalize(err error) error {
	var ce *httpx.CodeError
	if errors.As(err, &ce) {
		return ce
	}
	msg := err.Error()
	status := 500
	code := "internal_error"
	if strings.Contains(msg, "not found") {
		status = 404
		code = "not_found"
	} else if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "limit") || strings.Contains(msg, "parse") || strings.Contains(msg, "offset") || strings.Contains(msg, "capacity") || strings.Contains(msg, "must be") {
		status = 400
		code = "invalid_request"
	} else if strings.Contains(msg, "exists") || strings.Contains(msg, "status") {
		status = 409
		code = "conflict"
	}
	return &httpx.CodeError{Code: code, Message: msg, Status: status, Err: err}
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("invalid JSON trailing data")
	}
	return nil
}
