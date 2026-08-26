package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func RequestID(ctx context.Context) string { v, _ := ctx.Value(requestIDKey).(string); return v }

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
func RequestIDs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(context.Background(), requestIDKey, id)))
	})
}
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.Error("http panic recovered", "request_id", RequestID(r.Context()), "panic", v)
					WriteError(w, r, &CodeError{Code: "internal_error", Message: "internal server error", Status: 500})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":{"code":"timeout","message":"request timed out"}}`)
	}
}
func BodyLimit(max int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}
func Authenticate(apiKey string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "healthz") || strings.HasSuffix(r.URL.Path, "readyz") || strings.HasSuffix(r.URL.Path, "metrics") {
				next.ServeHTTP(w, r)
				return
			}
			if apiKey != "" && r.Header.Get("X-API-Key") != apiKey {
				WriteError(w, r, &CodeError{Code: "unauthorized", Message: "missing or invalid API key", Status: 401})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bucket struct {
	tokens float64
	last   time.Time
}
type Limiter struct {
	mu          sync.Mutex
	rate, burst float64
	clients     map[string]*bucket
}

func NewLimiter(rate, burst int) *Limiter {
	return &Limiter{rate: float64(rate), burst: float64(burst), clients: make(map[string]*bucket)}
}
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		now := time.Now()
		l.mu.Lock()
		b := l.clients[host]
		if b == nil {
			b = &bucket{tokens: l.burst, last: now}
			l.clients[host] = b
		}
		b.tokens += now.Sub(b.last).Seconds() * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
		allowed := b.tokens >= 1
		if allowed {
			b.tokens--
		}
		l.mu.Unlock()
		if !allowed {
			WriteError(w, r, &CodeError{Code: "rate_limited", Message: "request rate exceeded", Status: 429})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("http request", "method", r.Method, "path", r.URL.Path, "request_id", RequestID(r.Context()), "duration_ms", time.Since(start).Milliseconds())
		})
	}
}
