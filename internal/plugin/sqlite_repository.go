package plugin

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

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreatePackage(ctx context.Context, in CreatePackageInput) (PluginPackage, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.Manifest) == 0 {
		in.Manifest = json.RawMessage(`{}`)
	}
	if strings.TrimSpace(in.Visibility) == "" {
		in.Visibility = command.VisibilityPrivate
	}

	packageID := newID("pkg")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO plugin_packages(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			name, version, package_type, manifest_json, artifact_uri, status,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		packageID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		in.Name,
		in.Version,
		in.PackageType,
		string(in.Manifest),
		strings.TrimSpace(in.ArtifactURI),
		PackageStatusUploaded,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return PluginPackage{}, fmt.Errorf("insert plugin package: %w", err)
	}

	return r.GetPackageForAccess(ctx, in.Context, packageID)
}

func (r *SQLiteRepository) ListPackages(ctx context.Context, params PackageListParams) (PackageListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	baseFilter := `FROM plugin_packages p
		WHERE p.tenant_id = ? AND p.workspace_id = ?
		  AND (
		    p.owner_id = ?
		    OR p.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = p.tenant_id
		        AND a.workspace_id = p.workspace_id
		        AND a.resource_type = 'plugin_package'
		        AND a.resource_id = p.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = ?
		        AND (a.expires_at IS NULL OR a.expires_at >= ?)
		        AND EXISTS (
		          SELECT 1 FROM json_each(a.permissions) perm
		          WHERE UPPER(COALESCE(perm.value, '')) = 'READ'
		        )
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return PackageListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json, p.name, p.version, p.package_type, p.manifest_json, p.artifact_uri, p.status, p.created_at, p.updated_at
			 `+baseFilter+`
			   AND ((p.created_at < ?) OR (p.created_at = ? AND p.id < ?))
			 ORDER BY p.created_at DESC, p.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.UTC().Format(time.RFC3339Nano),
			cursorAt.UTC().Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return PackageListResult{}, fmt.Errorf("list plugin packages by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPackages(rows)
		if err != nil {
			return PackageListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return PackageListResult{}, fmt.Errorf("encode package cursor: %w", err)
			}
		}

		return PackageListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json, p.name, p.version, p.package_type, p.manifest_json, p.artifact_uri, p.status, p.created_at, p.updated_at
		 `+baseFilter+`
		 ORDER BY p.created_at DESC, p.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return PackageListResult{}, fmt.Errorf("list plugin packages by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPackages(rows)
	if err != nil {
		return PackageListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) `+baseFilter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return PackageListResult{}, fmt.Errorf("count plugin packages: %w", err)
	}

	return PackageListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *SQLiteRepository) GetPackageForAccess(ctx context.Context, req command.RequestContext, packageID string) (PluginPackage, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, version, package_type, manifest_json, artifact_uri, status, created_at, updated_at
		 FROM plugin_packages
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		packageID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPackage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PluginPackage{}, ErrPackageNotFound
	}
	if err != nil {
		return PluginPackage{}, fmt.Errorf("query plugin package: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) CreateInstall(ctx context.Context, in CreateInstallInput) (PluginInstall, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	installID := newID("ins")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO plugin_installs(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			package_id, scope, status, error_code, message_key, installed_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		installID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		command.VisibilityPrivate,
		"[]",
		in.PackageID,
		in.Scope,
		InstallStatusUploaded,
		nil,
		nil,
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return PluginInstall{}, fmt.Errorf("insert plugin install: %w", err)
	}

	return r.GetInstallForAccess(ctx, in.Context, installID)
}

func (r *SQLiteRepository) FindLatestPackageForUpgrade(ctx context.Context, in FindLatestPackageForUpgradeInput) (PluginPackage, error) {
	name := strings.TrimSpace(in.PackageName)
	if name == "" {
		return PluginPackage{}, ErrInvalidRequest
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, version, package_type, manifest_json, artifact_uri, status, created_at, updated_at
		 FROM plugin_packages
		 WHERE tenant_id = ? AND workspace_id = ? AND name = ? AND id <> ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		name,
		strings.TrimSpace(in.CurrentPackageID),
	)
	item, err := scanPackage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PluginPackage{}, ErrPackageNotFound
	}
	if err != nil {
		return PluginPackage{}, fmt.Errorf("query latest upgrade package: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) UpsertAlgorithms(ctx context.Context, in UpsertAlgorithmsInput) error {
	if len(in.Items) == 0 {
		return nil
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nowRaw := now.Format(time.RFC3339Nano)

	visibility := strings.TrimSpace(in.Visibility)
	if visibility == "" {
		visibility = command.VisibilityPrivate
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin algorithm upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range in.Items {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO algorithms(
				id, tenant_id, workspace_id, owner_id, visibility, acl_json,
				name, version, template_ref, defaults_json, constraints_json, dependencies_json, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				tenant_id=excluded.tenant_id,
				workspace_id=excluded.workspace_id,
				owner_id=excluded.owner_id,
				visibility=excluded.visibility,
				acl_json=excluded.acl_json,
				name=excluded.name,
				version=excluded.version,
				template_ref=excluded.template_ref,
				defaults_json=excluded.defaults_json,
				constraints_json=excluded.constraints_json,
				dependencies_json=excluded.dependencies_json,
				status=excluded.status,
				updated_at=excluded.updated_at`,
			item.ID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
			in.Context.OwnerID,
			visibility,
			"[]",
			item.Name,
			item.Version,
			item.TemplateRef,
			string(item.Defaults),
			string(item.Constraints),
			string(item.Dependencies),
			item.Status,
			nowRaw,
			nowRaw,
		); err != nil {
			return fmt.Errorf("upsert algorithm %s: %w", item.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit algorithm upsert tx: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) UpdateInstallStatus(ctx context.Context, in UpdateInstallStatusInput) (PluginInstall, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nowRaw := now.Format(time.RFC3339Nano)

	var (
		result sql.Result
		err    error
	)
	if in.Status == InstallStatusEnabled {
		var errorCode any
		var messageKey any
		if strings.TrimSpace(in.ErrorCode) != "" {
			errorCode = strings.TrimSpace(in.ErrorCode)
		}
		if strings.TrimSpace(in.MessageKey) != "" {
			messageKey = strings.TrimSpace(in.MessageKey)
		}
		result, err = r.db.ExecContext(
			ctx,
			`UPDATE plugin_installs
			 SET status = ?, error_code = ?, message_key = ?, installed_at = COALESCE(installed_at, ?), updated_at = ?
			 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
			in.Status,
			errorCode,
			messageKey,
			nowRaw,
			nowRaw,
			in.InstallID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
		)
	} else {
		var errorCode any
		var messageKey any
		if strings.TrimSpace(in.ErrorCode) != "" {
			errorCode = strings.TrimSpace(in.ErrorCode)
		}
		if strings.TrimSpace(in.MessageKey) != "" {
			messageKey = strings.TrimSpace(in.MessageKey)
		}
		result, err = r.db.ExecContext(
			ctx,
			`UPDATE plugin_installs
			 SET status = ?, error_code = ?, message_key = ?, updated_at = ?
			 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
			in.Status,
			errorCode,
			messageKey,
			nowRaw,
			in.InstallID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
		)
	}
	if err != nil {
		return PluginInstall{}, fmt.Errorf("update plugin install status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return PluginInstall{}, fmt.Errorf("update plugin install rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := r.GetInstallForAccess(ctx, in.Context, in.InstallID); err != nil {
			return PluginInstall{}, err
		}
	}

	return r.GetInstallForAccess(ctx, in.Context, in.InstallID)
}

func (r *SQLiteRepository) UpdateInstallPackage(ctx context.Context, in UpdateInstallPackageInput) (PluginInstall, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nowRaw := now.Format(time.RFC3339Nano)

	var errorCode any
	var messageKey any
	if strings.TrimSpace(in.ErrorCode) != "" {
		errorCode = strings.TrimSpace(in.ErrorCode)
	}
	if strings.TrimSpace(in.MessageKey) != "" {
		messageKey = strings.TrimSpace(in.MessageKey)
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE plugin_installs
		 SET package_id = ?, status = ?, error_code = ?, message_key = ?, installed_at = COALESCE(installed_at, ?), updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		in.PackageID,
		in.Status,
		errorCode,
		messageKey,
		nowRaw,
		nowRaw,
		in.InstallID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return PluginInstall{}, fmt.Errorf("update plugin install package: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PluginInstall{}, fmt.Errorf("update plugin install package rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := r.GetInstallForAccess(ctx, in.Context, in.InstallID); err != nil {
			return PluginInstall{}, err
		}
	}
	return r.GetInstallForAccess(ctx, in.Context, in.InstallID)
}

func (r *SQLiteRepository) CreateInstallHistory(ctx context.Context, in CreateInstallHistoryInput) (PluginInstallHistory, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	historyID := newID("phs")

	var errorCode any
	var messageKey any
	if strings.TrimSpace(in.ErrorCode) != "" {
		errorCode = strings.TrimSpace(in.ErrorCode)
	}
	if strings.TrimSpace(in.MessageKey) != "" {
		messageKey = strings.TrimSpace(in.MessageKey)
	}

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO plugin_install_history(
			id, tenant_id, workspace_id, install_id, from_version, to_version, command_id, status, error_code, message_key, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		historyID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.InstallID,
		strings.TrimSpace(in.FromVersion),
		strings.TrimSpace(in.ToVersion),
		strings.TrimSpace(in.CommandID),
		strings.TrimSpace(in.Status),
		errorCode,
		messageKey,
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return PluginInstallHistory{}, fmt.Errorf("insert plugin install history: %w", err)
	}

	return PluginInstallHistory{
		ID:          historyID,
		TenantID:    in.Context.TenantID,
		WorkspaceID: in.Context.WorkspaceID,
		InstallID:   in.InstallID,
		FromVersion: strings.TrimSpace(in.FromVersion),
		ToVersion:   strings.TrimSpace(in.ToVersion),
		CommandID:   strings.TrimSpace(in.CommandID),
		Status:      strings.TrimSpace(in.Status),
		ErrorCode:   strings.TrimSpace(in.ErrorCode),
		MessageKey:  strings.TrimSpace(in.MessageKey),
		CreatedAt:   now,
	}, nil
}

func (r *SQLiteRepository) GetInstallForAccess(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, package_id, scope, status, error_code, message_key, installed_at, created_at, updated_at
		 FROM plugin_installs
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		installID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanInstall(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PluginInstall{}, ErrInstallNotFound
	}
	if err != nil {
		return PluginInstall{}, fmt.Errorf("query plugin install: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) HasPermission(
	ctx context.Context,
	req command.RequestContext,
	resourceType string,
	resourceID string,
	permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(resourceID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = ?
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (
		     SELECT 1 FROM json_each(a.permissions) p
		     WHERE UPPER(COALESCE(p.value, '')) = ?
		   )
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		resourceType,
		resourceID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
		strings.ToUpper(strings.TrimSpace(permission)),
	)
	var marker int
	err := row.Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query plugin permission: %w", err)
	}
	return true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPackages(rows *sql.Rows) ([]PluginPackage, error) {
	items := make([]PluginPackage, 0)
	for rows.Next() {
		item, err := scanPackage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plugin packages: %w", err)
	}
	return items, nil
}

func scanPackage(row rowScanner) (PluginPackage, error) {
	var (
		item         PluginPackage
		aclRaw       string
		manifestRaw  string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.Version,
		&item.PackageType,
		&manifestRaw,
		&item.ArtifactURI,
		&item.Status,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return PluginPackage{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(manifestRaw) == "" {
		manifestRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.ManifestJSON = json.RawMessage(manifestRaw)

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return PluginPackage{}, fmt.Errorf("parse plugin package created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return PluginPackage{}, fmt.Errorf("parse plugin package updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanInstall(row rowScanner) (PluginInstall, error) {
	var (
		item           PluginInstall
		aclRaw         string
		errorCodeRaw   sql.NullString
		messageKeyRaw  sql.NullString
		installedAtRaw sql.NullString
		createdAtRaw   string
		updatedAtRaw   string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.PackageID,
		&item.Scope,
		&item.Status,
		&errorCodeRaw,
		&messageKeyRaw,
		&installedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return PluginInstall{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	if errorCodeRaw.Valid {
		item.ErrorCode = errorCodeRaw.String
	}
	if messageKeyRaw.Valid {
		item.MessageKey = messageKeyRaw.String
	}
	if installedAtRaw.Valid && strings.TrimSpace(installedAtRaw.String) != "" {
		installedAt, err := time.Parse(time.RFC3339Nano, installedAtRaw.String)
		if err != nil {
			return PluginInstall{}, fmt.Errorf("parse plugin install installed_at: %w", err)
		}
		item.InstalledAt = &installedAt
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return PluginInstall{}, fmt.Errorf("parse plugin install created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return PluginInstall{}, fmt.Errorf("parse plugin install updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}
