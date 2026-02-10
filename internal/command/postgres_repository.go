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

func (r *PostgresRepository) GetForAccess(context.Context, RequestContext, string) (Command, error) {
	return Command{}, ErrNotImplemented
}

func (r *PostgresRepository) List(context.Context, ListParams) (ListResult, error) {
	return ListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) HasCommandPermission(context.Context, RequestContext, string, string, time.Time) (bool, error) {
	return false, ErrNotImplemented
}

func (r *PostgresRepository) GetAssetForShare(context.Context, RequestContext, string) (ShareAsset, error) {
	return ShareAsset{}, ErrNotImplemented
}

func (r *PostgresRepository) HasAssetPermission(context.Context, RequestContext, string, string, time.Time) (bool, error) {
	return false, ErrNotImplemented
}

func (r *PostgresRepository) CreateShare(context.Context, ShareCreateInput) (Share, error) {
	return Share{}, ErrNotImplemented
}

func (r *PostgresRepository) ListShares(context.Context, ShareListParams) (ShareListResult, error) {
	return ShareListResult{}, ErrNotImplemented
}

func (r *PostgresRepository) DeleteShare(context.Context, RequestContext, string) error {
	return ErrNotImplemented
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
