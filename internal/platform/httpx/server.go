package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Server struct{ server *http.Server }

func NewServer(address string, handler http.Handler) *Server {
	return &Server{server: &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}}
}
func (s *Server) ListenAndServe() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http listen: %w", err)
	}
	return nil
}
func (s *Server) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
