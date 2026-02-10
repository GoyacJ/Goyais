package stream

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	StreamStatusOffline   = "offline"
	StreamStatusOnline    = "online"
	StreamStatusRecording = "recording"
	StreamStatusError     = "error"
)

const (
	RecordingStatusStarting  = "starting"
	RecordingStatusRecording = "recording"
	RecordingStatusStopping  = "stopping"
	RecordingStatusSucceeded = "succeeded"
	RecordingStatusFailed    = "failed"
	RecordingStatusCanceled  = "canceled"
)

const (
	ResourceTypeStream    = "streaming_asset"
	ResourceTypeRecording = "stream_recording"
)

type Stream struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage

	Path      string
	Protocol  string
	Source    string
	EndpointsJSON json.RawMessage
	StateJSON json.RawMessage
	Status    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Recording struct {
	ID          string
	StreamID    string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string

	Status     string
	AssetID    string
	ErrorCode  string
	MessageKey string

	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateStreamInput struct {
	Context command.RequestContext

	Path       string
	Protocol   string
	Source     string
	Visibility string
	State      json.RawMessage

	Now time.Time
}

type StreamListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type StreamListResult struct {
	Items      []Stream
	Total      int64
	NextCursor string
	UsedCursor bool
}

type CreateRecordingInput struct {
	Context  command.RequestContext
	StreamID string
	Now      time.Time
}

type CompleteRecordingInput struct {
	Context     command.RequestContext
	RecordingID string
	AssetID     string
	Now         time.Time
}

type UpdateStreamStatusInput struct {
	Context  command.RequestContext
	StreamID string
	Status   string
	State    json.RawMessage
	Now      time.Time
}

type CreateLineageInput struct {
	Context      command.RequestContext
	TargetAssetID string
	StepID       string
	Relation     string
	Now          time.Time
}

type StartRecordingResult struct {
	Stream             Stream
	Recording          Recording
	OnPublishTemplateID string
}

type StopRecordingResult struct {
	Stream    Stream
	Recording Recording
	AssetID   string
	LineageID string
}
