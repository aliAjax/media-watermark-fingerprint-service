package application

import (
	"context"
	audioapp "github.com/acme/media-watermark-fingerprinting/internal/audio/application"
	containerapp "github.com/acme/media-watermark-fingerprinting/internal/container/application"
	containerdomain "github.com/acme/media-watermark-fingerprinting/internal/container/domain"
	evidenceapp "github.com/acme/media-watermark-fingerprinting/internal/evidence/application"
	evidencedomain "github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
	fingerprintapp "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/application"
	fingerprintdomain "github.com/acme/media-watermark-fingerprinting/internal/fingerprint/domain"
	ingestapp "github.com/acme/media-watermark-fingerprinting/internal/ingest/application"
	jobapp "github.com/acme/media-watermark-fingerprinting/internal/job/application"
	jobdomain "github.com/acme/media-watermark-fingerprinting/internal/job/domain"
	"github.com/acme/media-watermark-fingerprinting/internal/platform/config"
	storageapp "github.com/acme/media-watermark-fingerprinting/internal/storage/application"
	videoapp "github.com/acme/media-watermark-fingerprinting/internal/video/application"
	watermarkapp "github.com/acme/media-watermark-fingerprinting/internal/watermark/application"
	watermarkdomain "github.com/acme/media-watermark-fingerprinting/internal/watermark/domain"
	workerapp "github.com/acme/media-watermark-fingerprinting/internal/worker/application"
	"time"
)

type EvidenceStore interface{ evidenceapp.Store }
type ConfigStore interface {
	Current(context.Context, string) (fingerprintdomain.AlgorithmConfig, error)
	Publish(context.Context, string) (fingerprintdomain.AlgorithmConfig, error)
	CreateDraft(context.Context, fingerprintdomain.AlgorithmConfig) (fingerprintdomain.AlgorithmConfig, error)
	Rollback(context.Context, string, int) (fingerprintdomain.AlgorithmConfig, error)
}
type KeyStore interface{ watermarkapp.KeyStore }
type App struct {
	Objects                 *storageapp.Service
	Containers              *containerapp.Registry
	Audio                   *audioapp.Service
	Video                   *videoapp.Service
	Fingerprints            fingerprintapp.Repository
	Matcher                 *fingerprintapp.Matcher
	Algorithms              ConfigStore
	Watermarks              *watermarkapp.Service
	Keys                    KeyStore
	Evidence                *evidenceapp.Service
	EvidenceStore           EvidenceStore
	Uploads                 *ingestapp.Service
	Jobs                    *jobapp.Queue
	Node                    *workerapp.Node
	assetIDs, fpIDs, jobIDs *config.IDs
	limits                  containerdomain.Limits
	started                 time.Time
}

func New(objects *storageapp.Service, containers *containerapp.Registry, audio *audioapp.Service, video *videoapp.Service, fps fingerprintapp.Repository, algorithms ConfigStore, watermarks *watermarkapp.Service, keys KeyStore, evidence *evidenceapp.Service, evidenceStore EvidenceStore, uploads *ingestapp.Service, jobs *jobapp.Queue, node *workerapp.Node, maxBytes int64) *App {
	a := &App{Objects: objects, Containers: containers, Audio: audio, Video: video, Fingerprints: fps, Matcher: fingerprintapp.NewMatcher(fps), Algorithms: algorithms, Watermarks: watermarks, Keys: keys, Evidence: evidence, EvidenceStore: evidenceStore, Uploads: uploads, Jobs: jobs, Node: node, assetIDs: config.NewIDs("asset"), fpIDs: config.NewIDs("fp"), jobIDs: config.NewIDs("job"), limits: containerdomain.DefaultLimits(maxBytes), started: time.Now().UTC()}
	a.registerJobs()
	return a
}
func (a *App) Ready() bool {
	return a.Objects != nil && a.Containers != nil && a.EvidenceStore != nil && a.Node.Snapshot().State != "stopped"
}
func (a *App) Uptime() time.Duration { return time.Since(a.started) }
func (a *App) Shutdown(ctx context.Context) error {
	_ = a.Jobs.Shutdown(context.Background())
	return nil
}
func (a *App) GetAsset(ctx context.Context, id string) (evidencedomain.Asset, error) {
	return a.EvidenceStore.GetAsset(ctx, id)
}
func (a *App) GetJob(ctx context.Context, id string) (jobdomain.Job, error) {
	return a.Jobs.Get(ctx, id)
}
func (a *App) DetectWatermark(ctx context.Context, assetID string) (watermarkdomain.Result, error) {
	asset, err := a.GetAsset(ctx, assetID)
	if err != nil {
		return watermarkdomain.Result{}, err
	}
	data, _, err := a.Objects.Bytes(ctx, asset.ObjectKey)
	if err != nil {
		return watermarkdomain.Result{}, err
	}
	return a.Watermarks.Detect(ctx, data)
}
