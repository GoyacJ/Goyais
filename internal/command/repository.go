package command

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, in CreateInput) (CreateResult, error)
	Get(ctx context.Context, req RequestContext, id string) (Command, error)
	List(ctx context.Context, params ListParams) (ListResult, error)
	AppendCommandEvent(ctx context.Context, req RequestContext, commandID, eventType string, payload []byte) error
	AppendAuditEvent(ctx context.Context, req RequestContext, commandID, eventType, decision, reason string, payload []byte) error
	SetStatus(ctx context.Context, req RequestContext, commandID, status string, result []byte, errorCode, messageKey string, finishedAt *time.Time) (Command, error)
}
