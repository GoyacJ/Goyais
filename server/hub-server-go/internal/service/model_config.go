package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
)

type ModelConfigSummary struct {
	ModelConfigID string  `json:"model_config_id"`
	WorkspaceID   string  `json:"workspace_id"`
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	BaseURL       *string `json:"base_url"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     *int    `json:"max_tokens"`
	SecretRef     string  `json:"secret_ref"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type CreateModelConfigInput struct {
	Provider    string
	Model       string
	BaseURL     *string
	Temperature *float64
	MaxTokens   *int
	APIKey      string
}

type UpdateModelConfigInput struct {
	Provider    *string
	Model       *string
	BaseURL     *string
	Temperature *float64
	MaxTokens   *int
	APIKey      *string
}

type ModelConfigService struct {
	db       *sql.DB
	authMode string
}

func NewModelConfigService(db *sql.DB, authMode string) *ModelConfigService {
	return &ModelConfigService{db: db, authMode: authMode}
}

func (s *ModelConfigService) List(ctx context.Context, workspaceID string) ([]ModelConfigSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT config_id, workspace_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at
		FROM model_configs
		WHERE workspace_id = ?
		ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ModelConfigSummary, 0)
	for rows.Next() {
		item, err := scanModelConfig(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *ModelConfigService) Get(ctx context.Context, workspaceID, modelConfigID string) (*ModelConfigSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT config_id, workspace_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at
		FROM model_configs
		WHERE config_id = ? AND workspace_id = ?`,
		modelConfigID, workspaceID)
	item, err := scanModelConfig(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func (s *ModelConfigService) Create(
	ctx context.Context,
	workspaceID string,
	in CreateModelConfigInput,
) (*ModelConfigSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}
	if strings.TrimSpace(in.Provider) == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(in.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(in.APIKey) == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	modelConfigID := uuid.NewString()
	temperature := 0.0
	if in.Temperature != nil {
		temperature = *in.Temperature
	}

	secretRef := fmt.Sprintf("keychain:%s:%s", strings.ToLower(strings.TrimSpace(in.Provider)), modelConfigID)
	if s.authMode != middleware.AuthModeLocalOpen {
		secretRef = fmt.Sprintf("secret:%s", modelConfigID)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO model_configs
			(config_id, workspace_id, display_name, provider, model, base_url, temperature, max_tokens, secret_ref, is_default, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)`,
		modelConfigID,
		workspaceID,
		fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(in.Provider)), strings.TrimSpace(in.Model)),
		strings.ToLower(strings.TrimSpace(in.Provider)),
		strings.TrimSpace(in.Model),
		normalizeOptionalString(in.BaseURL),
		temperature,
		in.MaxTokens,
		secretRef,
		user.UserID,
		now,
		now,
	); err != nil {
		return nil, err
	}

	if s.authMode != middleware.AuthModeLocalOpen {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO secrets(secret_ref, workspace_id, kind, value_encrypted, created_by, created_at)
			VALUES (?, ?, 'api_key', ?, ?, ?)`,
			secretRef, workspaceID, strings.TrimSpace(in.APIKey), user.UserID, now,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, modelConfigID)
}

func (s *ModelConfigService) Update(
	ctx context.Context,
	workspaceID, modelConfigID string,
	in UpdateModelConfigInput,
) (*ModelConfigSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}

	existing, err := s.Get(ctx, workspaceID, modelConfigID)
	if err != nil || existing == nil {
		return existing, err
	}

	provider := existing.Provider
	if in.Provider != nil && strings.TrimSpace(*in.Provider) != "" {
		provider = strings.ToLower(strings.TrimSpace(*in.Provider))
	}
	modelName := existing.Model
	if in.Model != nil && strings.TrimSpace(*in.Model) != "" {
		modelName = strings.TrimSpace(*in.Model)
	}

	baseURL := existing.BaseURL
	if in.BaseURL != nil {
		normalized := strings.TrimSpace(*in.BaseURL)
		if normalized == "" {
			baseURL = nil
		} else {
			baseURL = &normalized
		}
	}

	temperature := existing.Temperature
	if in.Temperature != nil {
		temperature = *in.Temperature
	}

	maxTokens := existing.MaxTokens
	if in.MaxTokens != nil {
		next := *in.MaxTokens
		maxTokens = &next
	}

	secretRef := existing.SecretRef
	if s.authMode == middleware.AuthModeLocalOpen {
		secretRef = fmt.Sprintf("keychain:%s:%s", provider, modelConfigID)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE model_configs SET
			display_name = ?,
			provider = ?,
			model = ?,
			base_url = ?,
			temperature = ?,
			max_tokens = ?,
			secret_ref = ?,
			updated_at = ?
		WHERE config_id = ? AND workspace_id = ?`,
		fmt.Sprintf("%s:%s", provider, modelName),
		provider,
		modelName,
		normalizeOptionalString(baseURL),
		temperature,
		maxTokens,
		secretRef,
		now,
		modelConfigID,
		workspaceID,
	); err != nil {
		return nil, err
	}

	if s.authMode != middleware.AuthModeLocalOpen && in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO secrets(secret_ref, workspace_id, kind, value_encrypted, created_by, created_at)
			VALUES (?, ?, 'api_key', ?, ?, ?)
			ON CONFLICT(secret_ref) DO UPDATE SET value_encrypted = excluded.value_encrypted`,
			secretRef,
			workspaceID,
			strings.TrimSpace(*in.APIKey),
			user.UserID,
			now,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, modelConfigID)
}

func (s *ModelConfigService) Delete(ctx context.Context, workspaceID, modelConfigID string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var secretRef string
	if err := tx.QueryRowContext(ctx,
		`SELECT secret_ref FROM model_configs WHERE config_id = ? AND workspace_id = ?`,
		modelConfigID, workspaceID,
	).Scan(&secretRef); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	result, err := tx.ExecContext(ctx,
		`DELETE FROM model_configs WHERE config_id = ? AND workspace_id = ?`,
		modelConfigID, workspaceID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}

	if strings.HasPrefix(secretRef, "secret:") {
		_, _ = tx.ExecContext(ctx,
			`DELETE FROM secrets WHERE secret_ref = ? AND workspace_id = ?`,
			secretRef, workspaceID,
		)
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

type modelConfigScanner interface {
	Scan(dest ...any) error
}

func scanModelConfig(row modelConfigScanner) (*ModelConfigSummary, error) {
	var item ModelConfigSummary
	var baseURL sql.NullString
	var maxTokens sql.NullInt64
	if err := row.Scan(
		&item.ModelConfigID,
		&item.WorkspaceID,
		&item.Provider,
		&item.Model,
		&baseURL,
		&item.Temperature,
		&maxTokens,
		&item.SecretRef,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if baseURL.Valid {
		item.BaseURL = &baseURL.String
	}
	if maxTokens.Valid {
		value := int(maxTokens.Int64)
		item.MaxTokens = &value
	}
	return &item, nil
}

func normalizeOptionalString(value *string) any {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return normalized
}
