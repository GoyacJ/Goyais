package stream

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	CreateStream(ctx context.Context, in CreateStreamInput) (Stream, error)
	GetStreamForAccess(ctx context.Context, req command.RequestContext, streamID string) (Stream, error)
	ListStreams(ctx context.Context, params StreamListParams) (StreamListResult, error)
	UpdateStreamStatus(ctx context.Context, in UpdateStreamStatusInput) (Stream, error)

	CreateRecording(ctx context.Context, in CreateRecordingInput) (Recording, error)
	GetActiveRecording(ctx context.Context, req command.RequestContext, streamID string) (Recording, error)
	CompleteRecording(ctx context.Context, in CompleteRecordingInput) (Recording, error)

	CreateLineage(ctx context.Context, in CreateLineageInput) (string, error)
	HasPermission(ctx context.Context, req command.RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported stream repository driver: %s", dbDriver)
	}
}
