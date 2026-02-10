package algorithm

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusCanceled  = "canceled"
)

type Run struct {
	ID            string
	TenantID      string
	WorkspaceID   string
	OwnerID       string
	Visibility    string
	ACLJSON       json.RawMessage
	AlgorithmID   string
	WorkflowRunID string
	CommandID     string
	OutputsJSON   json.RawMessage
	AssetIDsJSON  json.RawMessage
	Status        string
	ErrorCode     string
	MessageKey    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RunInput struct {
	Context     command.RequestContext
	AlgorithmID string
	Inputs      json.RawMessage
	Visibility  string
	Mode        string
}

type CreateRunInput struct {
	Context       command.RequestContext
	AlgorithmID   string
	WorkflowRunID string
	CommandID     string
	Visibility    string
	Outputs       json.RawMessage
	AssetIDs      []string
	Status        string
	ErrorCode     string
	MessageKey    string
	Now           time.Time
}
