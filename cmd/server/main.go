package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	audioadapter "github.com/acme/media-watermark-fingerprinting/internal/audio/adapter"
	audioapp "github.com/acme/media-watermark-fingerprinting/internal/audio/application"
	containeradapter "github.com/acme/media-watermark-fingerprinting/internal/container/adapter"
	containerapp "github.com/acme/media-watermark-fingerprinting/internal/container/application"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	evidenceapp "github.com/acme/media-watermark-fingerprinting/internal/evidence/application"
	evidencememory "github.com/acme/media-watermark-fingerprinting/internal/evidence/infrastructure"
	fpmemory "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/infrastructure"
	ingestapp "github.com/acme/media-watermark-fingerprinting/internal/ingest/application"
	ingestmemory "github.com/acme/media-watermark-fingerprinting/internal/ingest/infrastructure"
	jobapp "github.com/acme/media-watermark-fingerprinting/internal/job/application"
	jobmemory "github.com/acme/media-watermark-fingerprinting/internal/job/infrastructure"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/config"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/grpcx"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/httpx"
	serviceadapter "github.com/acme/media-watermark-fingerprinting/internal/service/adapter"
	serviceapp "github.com/acme/media-watermark-fingerprinting/internal/service/application"
	storageapp "github.com/acme/media-watermark-fingerprinting/internal/storage/application"
	storagememory "github.com/acme/media-watermark-fingerprinting/internal/storage/infrastructure"
	videoadapter "github.com/acme/media-watermark-fingerprinting/internal/video/adapter"
	videoapp "github.com/acme/media-watermark-fingerprinting/internal/video/application"
	watermarkadapter "github.com/acme/media-watermark-fingerprinting/internal/watermark/adapter"
	watermarkapp "github.com/acme/media-watermark-fingerprinting/internal/watermark/application"
	watermarkmemory "github.com/acme/media-watermark-fingerprinting/internal/watermark/infrastructure"
	workerapp "github.com/acme/media-watermark-fingerprinting/internal/worker/application"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "YAML configuration file")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("configuration rejected", "error", err)
		os.Exit(2)
	}
	application := build(cfg)
	handler := serviceadapter.NewHTTP(application)
	limiter := httpx.NewLimiter(30, 60)
	root := httpx.Chain(handler, httpx.RequestIDs, httpx.Recover(logger), httpx.Timeout(cfg.RequestTimeout), httpx.BodyLimit(cfg.MaxBodyBytes), httpx.Authenticate(cfg.APIKey), limiter.Middleware, httpx.Logging(logger))
	httpServer := httpx.NewServer(cfg.HTTPAddress, root)
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		logger.Error("gRPC listener failed", "error", err)
		os.Exit(2)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcx.Auth(cfg.APIKey)))
	grpcx.Register(grpcServer, grpcx.New(application))
	failures := make(chan error, 2)
	go func() {
		logger.Info("HTTP server started", "address", cfg.HTTPAddress)
		failures <- httpServer.ListenAndServe()
	}()
	go func() {
		logger.Info("gRPC server started", "address", cfg.GRPCAddress)
		failures <- grpcServer.Serve(grpcListener)
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-failures:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	application.Node.Drain()
	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}
	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Error("application shutdown failed", "error", err)
	}
	logger.Info("shutdown complete")
}
func build(c config.Config) *serviceapp.App {
	clock := config.Clock{}
	objectMemory := storagememory.NewMemory()
	objects := storageapp.New(objectMemory, c.MaxAssetBytes)
	containers := containerapp.NewRegistry()
	containers.Register(containerdomain.FormatWAV, containeradapter.WAVParser{})
	containers.Register(containerdomain.FormatMP4, containeradapter.MP4Parser{})
	containers.Register(containerdomain.FormatWebM, containeradapter.WebMParser{})
	containers.Register(containerdomain.FormatMPEGTS, containeradapter.TSParser{})
	containers.Register(containerdomain.FormatMP3, containeradapter.MP3Parser{})
	audio := audioapp.New(audioadapter.MockDecoder{}, 48_000*60*10)
	video := videoapp.New(videoadapter.MockDecoder{}, 300)
	fingerprints := fpmemory.NewMemory()
	algorithms := fpmemory.NewConfigStore()
	keys := watermarkmemory.NewKeys([]byte(c.WatermarkSecret))
	watermarks := watermarkapp.New(keys, watermarkadapter.LSB{}, watermarkapp.RealClock{})
	evidenceStore := evidencememory.NewMemory()
	evidence := evidenceapp.New(evidenceStore, clock)
	uploads := ingestapp.New(ingestmemory.NewMemory(), objects, clock, c.MaxAssetBytes)
	jobs := jobapp.NewQueue(jobmemory.NewMemory(), clock, c.WorkerCount, 3)
	node := workerapp.NewNode(hostname(), c.WorkerCount)
	return serviceapp.New(objects, containers, audio, video, fingerprints, algorithms, watermarks, keys, evidence, evidenceStore, uploads, jobs, node, c.MaxAssetBytes)
}
func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "media-node"
	}
	return name
}

var _ = fmt.Sprintf
