package command

import (
	"context"
	"database/sql"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(context.Context, CreateInput) (CreateResult, error) {
	return CreateResult{}, ErrNotImplemented
}

func (r *PostgresRepository) Get(context.Context, RequestContext, string) (Command, error) {
	return Command{}, ErrNotImplemented
}

func (r *PostgresRepository) List(context.Context, ListParams) (ListResult, error) {
	return ListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) AppendCommandEvent(context.Context, RequestContext, string, string, []byte) error {
	return ErrNotImplemented
}

func (r *PostgresRepository) AppendAuditEvent(context.Context, RequestContext, string, string, string, string, []byte) error {
	return ErrNotImplemented
}

func (r *PostgresRepository) SetStatus(context.Context, RequestContext, string, string, []byte, string, string, *time.Time) (Command, error) {
	return Command{}, ErrNotImplemented
}
