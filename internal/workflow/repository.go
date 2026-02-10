package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	CreateTemplate(ctx context.Context, in CreateTemplateInput) (WorkflowTemplate, error)
	PatchTemplate(ctx context.Context, in PatchTemplateInput) (WorkflowTemplate, error)
	PublishTemplate(ctx context.Context, in PublishTemplateInput) (WorkflowTemplate, error)
	GetTemplateForAccess(ctx context.Context, req command.RequestContext, templateID string) (WorkflowTemplate, error)
	ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResult, error)
	HasTemplatePermission(ctx context.Context, req command.RequestContext, templateID, permission string, now time.Time) (bool, error)

	CreateRun(ctx context.Context, in CreateRunInput) (WorkflowRun, error)
	CancelRun(ctx context.Context, in CancelRunInput) (WorkflowRun, error)
	GetRunForAccess(ctx context.Context, req command.RequestContext, runID string) (WorkflowRun, error)
	ListRuns(ctx context.Context, params RunListParams) (RunListResult, error)
	HasRunPermission(ctx context.Context, req command.RequestContext, runID, permission string, now time.Time) (bool, error)
	ListStepRuns(ctx context.Context, params StepListParams) (StepListResult, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported workflow repository driver: %s", dbDriver)
	}
}
