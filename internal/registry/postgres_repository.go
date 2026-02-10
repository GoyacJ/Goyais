package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetCapabilityForAccess(ctx context.Context, req command.RequestContext, capabilityID string) (Capability, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, provider_id, name, kind, version, input_schema::text, output_schema::text, required_permissions::text, egress_policy::text, status, created_at, updated_at
		 FROM capabilities
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		strings.TrimSpace(capabilityID),
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresCapability(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Capability{}, ErrCapabilityNotFound
	}
	if err != nil {
		return Capability{}, fmt.Errorf("query capability for access: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) GetAlgorithmForAccess(ctx context.Context, req command.RequestContext, algorithmID string) (Algorithm, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, name, version, template_ref, defaults_json::text, constraints_json::text, dependencies_json::text, status, created_at, updated_at
		 FROM algorithms
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		strings.TrimSpace(algorithmID),
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresAlgorithm(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Algorithm{}, ErrAlgorithmNotFound
	}
	if err != nil {
		return Algorithm{}, fmt.Errorf("query algorithm for access: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListCapabilities(ctx context.Context, params ListParams) (CapabilityListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC()
	resourceType := ResourceTypeCapability

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return CapabilityListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.provider_id, c.name, c.kind, c.version, c.input_schema::text, c.output_schema::text, c.required_permissions::text, c.egress_policy::text, c.status, c.created_at, c.updated_at
			 FROM capabilities c
			 `+postgresReadableFilter("c", 1)+`
			   AND ((c.created_at < $7) OR (c.created_at = $8 AND c.id < $9))
			 ORDER BY c.created_at DESC, c.id DESC
			 LIMIT $10`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return CapabilityListResult{}, fmt.Errorf("list capabilities by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresCapabilities(rows)
		if err != nil {
			return CapabilityListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return CapabilityListResult{}, fmt.Errorf("encode next capability cursor: %w", err)
			}
		}

		return CapabilityListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.provider_id, c.name, c.kind, c.version, c.input_schema::text, c.output_schema::text, c.required_permissions::text, c.egress_policy::text, c.status, c.created_at, c.updated_at
		 FROM capabilities c
		 `+postgresReadableFilter("c", 1)+`
		 ORDER BY c.created_at DESC, c.id DESC
		 LIMIT $7 OFFSET $8`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return CapabilityListResult{}, fmt.Errorf("list capabilities by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresCapabilities(rows)
	if err != nil {
		return CapabilityListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM capabilities c
		 `+postgresReadableFilter("c", 1),
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return CapabilityListResult{}, fmt.Errorf("count capabilities: %w", err)
	}

	return CapabilityListResult{Items: items, Total: total}, nil
}

func (r *PostgresRepository) ListAlgorithms(ctx context.Context, params ListParams) (AlgorithmListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC()
	resourceType := ResourceTypeAlgorithm

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return AlgorithmListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT a.id, a.tenant_id, a.workspace_id, a.owner_id, a.visibility, a.acl_json::text, a.name, a.version, a.template_ref, a.defaults_json::text, a.constraints_json::text, a.dependencies_json::text, a.status, a.created_at, a.updated_at
			 FROM algorithms a
			 `+postgresReadableFilter("a", 1)+`
			   AND ((a.created_at < $7) OR (a.created_at = $8 AND a.id < $9))
			 ORDER BY a.created_at DESC, a.id DESC
			 LIMIT $10`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return AlgorithmListResult{}, fmt.Errorf("list algorithms by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresAlgorithms(rows)
		if err != nil {
			return AlgorithmListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return AlgorithmListResult{}, fmt.Errorf("encode next algorithm cursor: %w", err)
			}
		}

		return AlgorithmListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT a.id, a.tenant_id, a.workspace_id, a.owner_id, a.visibility, a.acl_json::text, a.name, a.version, a.template_ref, a.defaults_json::text, a.constraints_json::text, a.dependencies_json::text, a.status, a.created_at, a.updated_at
		 FROM algorithms a
		 `+postgresReadableFilter("a", 1)+`
		 ORDER BY a.created_at DESC, a.id DESC
		 LIMIT $7 OFFSET $8`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return AlgorithmListResult{}, fmt.Errorf("list algorithms by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresAlgorithms(rows)
	if err != nil {
		return AlgorithmListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM algorithms a
		 `+postgresReadableFilter("a", 1),
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return AlgorithmListResult{}, fmt.Errorf("count algorithms: %w", err)
	}

	return AlgorithmListResult{Items: items, Total: total}, nil
}

func (r *PostgresRepository) ListProviders(ctx context.Context, params ListParams) (ProviderListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC()
	resourceType := ResourceTypeCapabilityProvider

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ProviderListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json::text, p.name, p.provider_type, p.endpoint, p.metadata_json::text, p.status, p.created_at, p.updated_at
			 FROM capability_providers p
			 `+postgresReadableFilter("p", 1)+`
			   AND ((p.created_at < $7) OR (p.created_at = $8 AND p.id < $9))
			 ORDER BY p.created_at DESC, p.id DESC
			 LIMIT $10`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return ProviderListResult{}, fmt.Errorf("list providers by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresProviders(rows)
		if err != nil {
			return ProviderListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ProviderListResult{}, fmt.Errorf("encode next provider cursor: %w", err)
			}
		}

		return ProviderListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json::text, p.name, p.provider_type, p.endpoint, p.metadata_json::text, p.status, p.created_at, p.updated_at
		 FROM capability_providers p
		 `+postgresReadableFilter("p", 1)+`
		 ORDER BY p.created_at DESC, p.id DESC
		 LIMIT $7 OFFSET $8`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return ProviderListResult{}, fmt.Errorf("list providers by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresProviders(rows)
	if err != nil {
		return ProviderListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM capability_providers p
		 `+postgresReadableFilter("p", 1),
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return ProviderListResult{}, fmt.Errorf("count providers: %w", err)
	}

	return ProviderListResult{Items: items, Total: total}, nil
}

func (r *PostgresRepository) HasPermission(ctx context.Context, req command.RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(resourceType) == "" || strings.TrimSpace(resourceID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}

	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = $3
		   AND a.resource_id = $4
		   AND a.subject_type = 'user'
		   AND a.subject_id = $5
		   AND (a.expires_at IS NULL OR a.expires_at >= $6)
		   AND a.permissions @> jsonb_build_array($7::text)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		strings.TrimSpace(resourceType),
		strings.TrimSpace(resourceID),
		req.UserID,
		now.UTC(),
		strings.ToUpper(strings.TrimSpace(permission)),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query registry acl permission: %w", err)
	}
	return true, nil
}

func postgresReadableFilter(alias string, start int) string {
	alias = strings.TrimSpace(alias)
	tenant := start
	workspace := start + 1
	owner := start + 2
	resourceType := start + 3
	subject := start + 4
	now := start + 5

	return fmt.Sprintf(`WHERE %s.tenant_id = $%d AND %s.workspace_id = $%d
	  AND (
	    %s.owner_id = $%d
	    OR %s.visibility = 'WORKSPACE'
	    OR EXISTS (
	      SELECT 1 FROM acl_entries acl
	      WHERE acl.tenant_id = %s.tenant_id
	        AND acl.workspace_id = %s.workspace_id
	        AND acl.resource_type = $%d
	        AND acl.resource_id = %s.id
	        AND acl.subject_type = 'user'
	        AND acl.subject_id = $%d
	        AND (acl.expires_at IS NULL OR acl.expires_at >= $%d)
	        AND acl.permissions @> jsonb_build_array('READ')
	    )
	  )`, alias, tenant, alias, workspace, alias, owner, alias, alias, alias, resourceType, alias, subject, now)
}

func scanPostgresCapabilities(rows *sql.Rows) ([]Capability, error) {
	items := make([]Capability, 0)
	for rows.Next() {
		item, err := scanPostgresCapability(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate capabilities: %w", err)
	}
	return items, nil
}

func scanPostgresCapability(row interface{ Scan(dest ...any) error }) (Capability, error) {
	var item Capability
	var providerID sql.NullString
	var aclRaw string
	var inputSchemaRaw string
	var outputSchemaRaw string
	var requiredPermissionsRaw string
	var egressPolicyRaw string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&providerID,
		&item.Name,
		&item.Kind,
		&item.Version,
		&inputSchemaRaw,
		&outputSchemaRaw,
		&requiredPermissionsRaw,
		&egressPolicyRaw,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Capability{}, err
	}
	if providerID.Valid {
		item.ProviderID = providerID.String
	}
	item.ACLJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.InputSchemaJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(inputSchemaRaw)), "{}")
	item.OutputSchemaJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(outputSchemaRaw)), "{}")
	item.RequiredPermissionsJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(requiredPermissionsRaw)), "[]")
	item.EgressPolicyJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(egressPolicyRaw)), "{}")
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}

func scanPostgresAlgorithms(rows *sql.Rows) ([]Algorithm, error) {
	items := make([]Algorithm, 0)
	for rows.Next() {
		item, err := scanPostgresAlgorithm(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate algorithms: %w", err)
	}
	return items, nil
}

func scanPostgresAlgorithm(row interface{ Scan(dest ...any) error }) (Algorithm, error) {
	var item Algorithm
	var aclRaw string
	var defaultsRaw string
	var constraintsRaw string
	var dependenciesRaw string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.Version,
		&item.TemplateRef,
		&defaultsRaw,
		&constraintsRaw,
		&dependenciesRaw,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Algorithm{}, err
	}
	item.ACLJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.DefaultsJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(defaultsRaw)), "{}")
	item.ConstraintsJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(constraintsRaw)), "{}")
	item.DependenciesJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(dependenciesRaw)), "{}")
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}

func scanPostgresProviders(rows *sql.Rows) ([]CapabilityProvider, error) {
	items := make([]CapabilityProvider, 0)
	for rows.Next() {
		item, err := scanPostgresProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return items, nil
}

func scanPostgresProvider(row interface{ Scan(dest ...any) error }) (CapabilityProvider, error) {
	var item CapabilityProvider
	var aclRaw string
	var metadataRaw string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.ProviderType,
		&item.Endpoint,
		&metadataRaw,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return CapabilityProvider{}, err
	}
	item.ACLJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.MetadataJSON = ensureJSONRaw(json.RawMessage(strings.TrimSpace(metadataRaw)), "{}")
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}

func ensureJSONRaw(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}
