package domain

import "time"

type AssetStatus string

const (
	AssetActive    AssetStatus = "active"
	AssetFrozen    AssetStatus = "frozen"
	AssetWithdrawn AssetStatus = "withdrawn"
)

type Asset struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	ObjectKey string      `json:"object_key"`
	SHA256    string      `json:"sha256"`
	Size      int64       `json:"size"`
	Format    string      `json:"format"`
	Status    AssetStatus `json:"status"`
	Metadata  any         `json:"metadata,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
type Event struct {
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	Type      string    `json:"type"`
	Summary   string    `json:"summary"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	Withdrawn bool      `json:"withdrawn"`
}
type Review struct {
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	MatchID   string    `json:"match_id"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}
