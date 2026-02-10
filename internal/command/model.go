package command

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const (
	StatusAccepted  = "accepted"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"
)

const (
	VisibilityPrivate   = "PRIVATE"
	VisibilityWorkspace = "WORKSPACE"
	VisibilityTenant    = "TENANT"
	VisibilityPublic    = "PUBLIC"
)

const (
	PermissionRead    = "READ"
	PermissionWrite   = "WRITE"
	PermissionExecute = "EXECUTE"
	PermissionManage  = "MANAGE"
	PermissionShare   = "SHARE"
)

type RequestContext struct {
	TenantID    string
	WorkspaceID string
	UserID      string
	OwnerID     string
}

type Command struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage
	CommandType string
	Payload     json.RawMessage
	Status      string
	Result      json.RawMessage
	ErrorCode   string
	MessageKey  string
	AcceptedAt  time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateInput struct {
	Context        RequestContext
	CommandType    string
	Payload        json.RawMessage
	Visibility     string
	IdempotencyKey string
	RequestHash    string
	Now            time.Time
	TTL            time.Duration
}

type CreateResult struct {
	Command Command
	Reused  bool
}

type ListParams struct {
	Context  RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type ListResult struct {
	Items      []Command
	Total      int64
	NextCursor string
	UsedCursor bool
}

type Share struct {
	ID           string
	TenantID     string
	WorkspaceID  string
	ResourceType string
	ResourceID   string
	SubjectType  string
	SubjectID    string
	Permissions  []string
	ExpiresAt    *time.Time
	CreatedBy    string
	CreatedAt    time.Time
}

type ShareCreateInput struct {
	Context      RequestContext
	ResourceType string
	ResourceID   string
	SubjectType  string
	SubjectID    string
	Permissions  []string
	ExpiresAt    *time.Time
	Now          time.Time
}

type ShareListParams struct {
	Context  RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type ShareListResult struct {
	Items      []Share
	Total      int64
	NextCursor string
	UsedCursor bool
}

type cursorToken struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func EncodeCursor(t time.Time, id string) (string, error) {
	raw, err := json.Marshal(cursorToken{
		CreatedAt: t.UTC().Format(time.RFC3339Nano),
		ID:        id,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodeCursor(raw string) (time.Time, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("decode cursor: %w", err)
	}

	var tok cursorToken
	if err := json.Unmarshal(decoded, &tok); err != nil {
		return time.Time{}, "", fmt.Errorf("unmarshal cursor: %w", err)
	}

	if tok.ID == "" || tok.CreatedAt == "" {
		return time.Time{}, "", fmt.Errorf("cursor missing fields")
	}

	ts, err := time.Parse(time.RFC3339Nano, tok.CreatedAt)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse cursor time: %w", err)
	}

	return ts, tok.ID, nil
}
