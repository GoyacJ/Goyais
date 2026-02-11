package asset

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	StatusReady   = "ready"
	StatusDeleted = "deleted"
)

type Asset struct {
	ID           string
	TenantID     string
	WorkspaceID  string
	OwnerID      string
	Visibility   string
	ACLJSON      json.RawMessage
	Name         string
	Type         string
	Mime         string
	Size         int64
	URI          string
	Hash         string
	MetadataJSON json.RawMessage
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateInput struct {
	Context    command.RequestContext
	Name       string
	Type       string
	Mime       string
	Size       int64
	URI        string
	Hash       string
	Visibility string
	Metadata   json.RawMessage
	Now        time.Time
}

type UpdateInput struct {
	Context     command.RequestContext
	AssetID     string
	Name        *string
	Visibility  *string
	Metadata    json.RawMessage
	MetadataSet bool
	Now         time.Time
}

type LineageEdge struct {
	ID            string
	TenantID      string
	WorkspaceID   string
	SourceAssetID string
	TargetAssetID string
	RunID         string
	StepID        string
	Relation      string
	CreatedAt     time.Time
}

type ListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type ListResult struct {
	Items      []Asset
	Total      int64
	NextCursor string
	UsedCursor bool
}
