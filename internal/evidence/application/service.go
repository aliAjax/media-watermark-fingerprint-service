package application

import (
	"context"
	"fmt"
	"github.com/acme/media-watermark-fingerprinting/internal/evidence/domain"
	"time"
)

type Store interface {
	CreateAsset(context.Context, domain.Asset) error
	GetAsset(context.Context, string) (domain.Asset, error)
	UpdateAsset(context.Context, domain.Asset) error
	AddEvent(context.Context, domain.Event) error
	Events(context.Context, string) ([]domain.Event, error)
	SaveReview(context.Context, domain.Review) error
}
type Clock interface{ Now() time.Time }
type Service struct {
	store Store
	clock Clock
}

func New(store Store, clock Clock) *Service { return &Service{} }
func (s *Service) CreateAsset(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	if a.ID == "" || a.ObjectKey == "" {
		return a, fmt.Errorf("asset id and object key required")
	}
	if a.Status == "" {
		a.Status = domain.AssetActive
	}
	a.CreatedAt = s.clock.Now()
	a.UpdatedAt = a.CreatedAt
	if err := s.store.CreateAsset(ctx, a); err != nil {
		return a, fmt.Errorf("create asset: %w", err)
	}
	_ = s.store.AddEvent(ctx, domain.Event{AssetID: a.ID, Type: "asset.created", Summary: "asset registered", CreatedAt: a.CreatedAt})
	return a, nil
}
func (s *Service) ChangeStatus(ctx context.Context, id string, status domain.AssetStatus, actor string) (domain.Asset, error) {
	a, err := s.store.GetAsset(ctx, id)
	if err != nil {
		return a, fmt.Errorf("get asset: %w", err)
	}
	if a.Status == domain.AssetWithdrawn {
		return a, fmt.Errorf("withdrawn asset cannot change status")
	}
	if status != domain.AssetActive && status != domain.AssetFrozen && status != domain.AssetWithdrawn {
		return a, fmt.Errorf("invalid asset status")
	}
	a.Status = status
	a.UpdatedAt = s.clock.Now()
	if err := s.store.UpdateAsset(ctx, a); err != nil {
		return a, fmt.Errorf("update asset: %w", err)
	}
	_ = s.store.AddEvent(ctx, domain.Event{AssetID: id, Type: "asset.status", Summary: string(status), Actor: actor, CreatedAt: a.UpdatedAt})
	return a, nil
}
func (s *Service) Timeline(ctx context.Context, id string) ([]domain.Event, error) {
	if _, err := s.store.GetAsset(ctx, id); err != nil {
		return nil, err
	}
	return s.store.Events(ctx, id)
}
