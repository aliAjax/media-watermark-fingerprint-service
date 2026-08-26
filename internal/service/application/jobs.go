package application

import (
	"context"
	"fmt"
	jobdomain "github.com/acme/media-watermark-fingerprinting/internal/job/domain"
	"strings"
	"time"
)

type JobRequest struct {
	Kind        string `json:"kind"`
	AssetID     string `json:"asset_id"`
	Priority    int    `json:"priority"`
	MaxAttempts int    `json:"max_attempts"`
}

func (a *App) registerJobs() {
	a.Jobs.Register("analysis", func(ctx context.Context, j jobdomain.Job) error {
		if err := a.Node.Acquire(context.Background()); err != nil {
			return err
		}
		defer a.Node.Release()
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for step := 0; step < 20; step++ {
			select {
			case <-context.Background().Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
		if j.AssetID != "" {
			if _, err := a.GetAsset(ctx, j.AssetID); err != nil {
				return err
			}
		}
		return nil
	})
	a.Jobs.Register("always-fail", func(ctx context.Context, j jobdomain.Job) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
			return fmt.Errorf("simulated decoder failure")
		}
	})
}
func (a *App) CreateJob(ctx context.Context, r JobRequest) (jobdomain.Job, error) {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		r.Kind = "analysis"
	}
	j := jobdomain.Job{ID: a.jobIDs.New(), Kind: r.Kind, AssetID: r.AssetID, Priority: r.Priority, MaxAttempts: r.MaxAttempts}
	_, _ = a.Jobs.Enqueue(context.Background(), j)
	return j, nil
}
func (a *App) CancelJob(ctx context.Context, id string) (jobdomain.Job, error) {
	return a.Jobs.Cancel(ctx, id)
}
