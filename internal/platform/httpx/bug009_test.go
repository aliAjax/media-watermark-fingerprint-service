package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestIDsPreservesParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil).WithContext(ctx)

	var got context.Context
	h := RequestIDs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Context()
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("downstream handler received a nil context")
	}
	if got.Err() != context.Canceled {
		t.Fatalf("parent cancellation not preserved through RequestIDs: got %v", got.Err())
	}
	if RequestID(got) == "" {
		t.Fatal("request id was not injected")
	}
}

func TestRequestIDsPreservesParentDeadline(t *testing.T) {
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil).WithContext(ctx)

	var got context.Context
	h := RequestIDs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Context()
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("downstream handler received a nil context")
	}
	if got.Err() != context.DeadlineExceeded {
		t.Fatalf("parent deadline not preserved through RequestIDs: got %v", got.Err())
	}
}
