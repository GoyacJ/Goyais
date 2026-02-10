package asset

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

func (r *PostgresRepository) Create(context.Context, CreateInput) (Asset, error) {
	return Asset{}, ErrNotImplemented
}

func (r *PostgresRepository) GetForAccess(context.Context, command.RequestContext, string) (Asset, error) {
	return Asset{}, ErrNotImplemented
}

func (r *PostgresRepository) List(context.Context, ListParams) (ListResult, error) {
	return ListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) HasPermission(context.Context, command.RequestContext, string, string, time.Time) (bool, error) {
	return false, ErrNotImplemented
}
