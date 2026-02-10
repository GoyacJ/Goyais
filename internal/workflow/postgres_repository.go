package workflow

import (
	"context"
	"database/sql"
	"time"

	"goyais/internal/command"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateTemplate(context.Context, CreateTemplateInput) (WorkflowTemplate, error) {
	return WorkflowTemplate{}, ErrNotImplemented
}

func (r *PostgresRepository) PatchTemplate(context.Context, PatchTemplateInput) (WorkflowTemplate, error) {
	return WorkflowTemplate{}, ErrNotImplemented
}

func (r *PostgresRepository) PublishTemplate(context.Context, PublishTemplateInput) (WorkflowTemplate, error) {
	return WorkflowTemplate{}, ErrNotImplemented
}

func (r *PostgresRepository) GetTemplateForAccess(context.Context, command.RequestContext, string) (WorkflowTemplate, error) {
	return WorkflowTemplate{}, ErrNotImplemented
}

func (r *PostgresRepository) ListTemplates(context.Context, TemplateListParams) (TemplateListResult, error) {
	return TemplateListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) HasTemplatePermission(context.Context, command.RequestContext, string, string, time.Time) (bool, error) {
	return false, ErrNotImplemented
}

func (r *PostgresRepository) CreateRun(context.Context, CreateRunInput) (WorkflowRun, error) {
	return WorkflowRun{}, ErrNotImplemented
}

func (r *PostgresRepository) CancelRun(context.Context, CancelRunInput) (WorkflowRun, error) {
	return WorkflowRun{}, ErrNotImplemented
}

func (r *PostgresRepository) GetRunForAccess(context.Context, command.RequestContext, string) (WorkflowRun, error) {
	return WorkflowRun{}, ErrNotImplemented
}

func (r *PostgresRepository) ListRuns(context.Context, RunListParams) (RunListResult, error) {
	return RunListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) HasRunPermission(context.Context, command.RequestContext, string, string, time.Time) (bool, error) {
	return false, ErrNotImplemented
}

func (r *PostgresRepository) ListStepRuns(context.Context, StepListParams) (StepListResult, error) {
	return StepListResult{}, ErrNotImplemented
}
