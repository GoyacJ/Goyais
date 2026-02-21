package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db          *sql.DB
	tokenExpiry time.Duration
}

func NewAuthService(db *sql.DB, tokenExpiryHours int) *AuthService {
	return &AuthService{
		db:          db,
		tokenExpiry: time.Duration(tokenExpiryHours) * time.Hour,
	}
}

const (
	LocalOpenDefaultUserID      = "local-user"
	LocalOpenDefaultWorkspaceID = "local"
	localOpenDefaultEmail       = "local@goyais.local"
	localOpenDefaultDisplayName = "Local User"
	localOpenDefaultRoleName    = "Owner"
)

func (s *AuthService) IsBootstrapComplete(ctx context.Context) (bool, error) {
	var done int
	err := s.db.QueryRowContext(ctx,
		`SELECT setup_completed FROM system_state WHERE singleton_id = 1`).Scan(&done)
	if err != nil {
		return false, err
	}
	return done == 1, nil
}

// EnsureLocalOpenBootstrap guarantees local_open mode has a default principal and workspace.
func (s *AuthService) EnsureLocalOpenBootstrap(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRowContext(ctx,
		`SELECT user_id FROM users WHERE email = ? LIMIT 1`,
		localOpenDefaultEmail,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO users(user_id,email,password_hash,display_name,status,created_at)
			VALUES(?,?,?,?,? ,?)`,
			LocalOpenDefaultUserID,
			localOpenDefaultEmail,
			"local_open",
			localOpenDefaultDisplayName,
			"active",
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("create local user: %w", err)
		}
		userID = LocalOpenDefaultUserID
	} else if err != nil {
		return err
	}

	var workspaceID string
	err = tx.QueryRowContext(ctx,
		`SELECT workspace_id FROM workspaces WHERE slug = 'local' LIMIT 1`,
	).Scan(&workspaceID)
	if err == sql.ErrNoRows {
		workspaceID = LocalOpenDefaultWorkspaceID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO workspaces(workspace_id,name,slug,kind,created_by,created_at)
			VALUES(?,?,?,?,?,?)`,
			workspaceID,
			"Local Workspace",
			"local",
			"local",
			userID,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("create local workspace: %w", err)
		}
	} else if err != nil {
		return err
	}

	var roleID string
	err = tx.QueryRowContext(ctx,
		`SELECT role_id FROM roles WHERE workspace_id = ? AND name = ? LIMIT 1`,
		workspaceID,
		localOpenDefaultRoleName,
	).Scan(&roleID)
	if err == sql.ErrNoRows {
		roleID = uuid.NewString()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO roles(role_id,workspace_id,name,is_system,created_at) VALUES(?,?,?,?,?)`,
			roleID,
			workspaceID,
			localOpenDefaultRoleName,
			1,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("create local owner role: %w", err)
		}
	} else if err != nil {
		return err
	}

	permRows, err := tx.QueryContext(ctx, `SELECT perm_key FROM permissions`)
	if err != nil {
		return err
	}
	var permKeys []string
	for permRows.Next() {
		var permKey string
		if err := permRows.Scan(&permKey); err != nil {
			permRows.Close()
			return err
		}
		permKeys = append(permKeys, permKey)
	}
	if err := permRows.Close(); err != nil {
		return err
	}
	for _, permKey := range permKeys {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO role_permissions(role_id,perm_key) VALUES(?,?)`,
			roleID, permKey,
		); err != nil {
			return fmt.Errorf("grant role permission %s: %w", permKey, err)
		}
	}

	menuRows, err := tx.QueryContext(ctx, `SELECT menu_id FROM menus`)
	if err != nil {
		return err
	}
	var menuIDs []string
	for menuRows.Next() {
		var menuID string
		if err := menuRows.Scan(&menuID); err != nil {
			menuRows.Close()
			return err
		}
		menuIDs = append(menuIDs, menuID)
	}
	if err := menuRows.Close(); err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO role_menus(role_id,menu_id) VALUES(?,?)`,
			roleID, menuID,
		); err != nil {
			return fmt.Errorf("grant role menu %s: %w", menuID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO workspace_members(workspace_id,user_id,role_id,status,joined_at)
		VALUES(?,?,?,?,?)`,
		workspaceID,
		userID,
		roleID,
		"active",
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("ensure local membership: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE system_state SET setup_completed = 1, updated_at = ? WHERE singleton_id = 1`,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("mark setup completed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ResolveLocalOpenUser returns the injected local principal for local_open auth mode.
func (s *AuthService) ResolveLocalOpenUser(ctx context.Context) (*model.AuthUser, error) {
	user, err := s.loadAuthUserByID(ctx, LocalOpenDefaultUserID)
	if err == nil {
		return user, nil
	}
	var userID string
	if queryErr := s.db.QueryRowContext(ctx,
		`SELECT user_id FROM users WHERE email = ? LIMIT 1`,
		localOpenDefaultEmail,
	).Scan(&userID); queryErr != nil {
		return nil, queryErr
	}
	return s.loadAuthUserByID(ctx, userID)
}

func (s *AuthService) BootstrapAdmin(ctx context.Context, email, password, name string) (string, error) {
	done, err := s.IsBootstrapComplete(ctx)
	if err != nil {
		return "", err
	}
	if done {
		return "", fmt.Errorf("already bootstrapped")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	userID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Create user
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users(user_id,email,password_hash,display_name,status,created_at) VALUES(?,?,?,?,?,?)`,
		userID, email, string(hash), name, "active", now); err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	// Create default local workspace
	wsID := uuid.NewString()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO workspaces(workspace_id,name,slug,created_by,created_at) VALUES(?,?,?,?,?)`,
		wsID, "Local", "local", userID, now); err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}

	// Create Owner role
	roleID := uuid.NewString()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO roles(role_id,workspace_id,name,is_system,created_at) VALUES(?,?,?,?,?)`,
		roleID, wsID, "Owner", 1, now); err != nil {
		return "", fmt.Errorf("create role: %w", err)
	}

	// Grant all permissions to Owner
	rows, err := tx.QueryContext(ctx, `SELECT perm_key FROM permissions`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var pkey string
		if err := rows.Scan(&pkey); err != nil {
			return "", err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO role_permissions(role_id,perm_key) VALUES(?,?)`,
			roleID, pkey); err != nil {
			return "", err
		}
	}

	// Grant all menus to Owner
	menuRows, err := tx.QueryContext(ctx, `SELECT menu_id FROM menus`)
	if err != nil {
		return "", err
	}
	defer menuRows.Close()
	for menuRows.Next() {
		var mid string
		if err := menuRows.Scan(&mid); err != nil {
			return "", err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO role_menus(role_id,menu_id) VALUES(?,?)`,
			roleID, mid); err != nil {
			return "", err
		}
	}

	// Add user as workspace member
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO workspace_members(workspace_id,user_id,role_id,status,joined_at) VALUES(?,?,?,?,?)`,
		wsID, userID, roleID, "active", now); err != nil {
		return "", fmt.Errorf("add member: %w", err)
	}

	// Mark bootstrap complete
	if _, err := tx.ExecContext(ctx,
		`UPDATE system_state SET setup_completed=1, updated_at=? WHERE singleton_id=1`, now); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return s.issueToken(ctx, userID)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	var userID, hash string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, password_hash FROM users WHERE email=? AND status='active'`, email).
		Scan(&userID, &hash)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}
	return s.issueToken(ctx, userID)
}

func (s *AuthService) Logout(ctx context.Context, authHeader string) error {
	raw := strings.TrimPrefix(authHeader, "Bearer ")
	tokenHash := hashToken(raw)
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE token_hash=?`, tokenHash)
	return err
}

func (s *AuthService) Me(ctx context.Context) (map[string]any, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT wm.workspace_id, w.name, w.slug, r.name
		FROM workspace_members wm
		JOIN workspaces w ON w.workspace_id = wm.workspace_id
		JOIN roles r ON r.role_id = wm.role_id
		WHERE wm.user_id = ? AND wm.status = 'active'
		ORDER BY w.name`,
		user.UserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := make([]map[string]any, 0)
	for rows.Next() {
		var workspaceID, workspaceName, workspaceSlug, roleName string
		if err := rows.Scan(&workspaceID, &workspaceName, &workspaceSlug, &roleName); err != nil {
			return nil, err
		}
		memberships = append(memberships, map[string]any{
			"workspace_id":   workspaceID,
			"workspace_name": workspaceName,
			"workspace_slug": workspaceSlug,
			"role_name":      roleName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"user": map[string]any{
			"user_id":      user.UserID,
			"email":        user.Email,
			"display_name": user.DisplayName,
		},
		"memberships": memberships,
	}, nil
}

func (s *AuthService) Navigation(ctx context.Context, workspaceID string) (map[string]any, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	if strings.TrimSpace(workspaceID) == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	menuRows, err := s.db.QueryContext(ctx, `
		SELECT m.menu_id, m.parent_id, m.sort_order, m.route, m.icon_key, m.i18n_key
		FROM menus m
		JOIN role_menus rm ON rm.menu_id = m.menu_id
		JOIN workspace_members wm ON wm.role_id = rm.role_id
		WHERE wm.workspace_id = ? AND wm.user_id = ? AND wm.status = 'active'
		ORDER BY m.sort_order`, workspaceID, user.UserID)
	if err != nil {
		return nil, err
	}
	defer menuRows.Close()

	type menuNode struct {
		menuID    string
		parentID  *string
		sortOrder int
		payload   map[string]any
	}
	nodes := make(map[string]*menuNode)
	order := make([]*menuNode, 0)
	for menuRows.Next() {
		var menuID, route, iconKey, i18nKey string
		var parentID sql.NullString
		var sortOrder int
		if err := menuRows.Scan(&menuID, &parentID, &sortOrder, &route, &iconKey, &i18nKey); err != nil {
			return nil, err
		}
		node := &menuNode{
			menuID:    menuID,
			sortOrder: sortOrder,
			payload: map[string]any{
				"menu_id":  menuID,
				"route":    nullIfEmpty(route),
				"icon_key": nullIfEmpty(iconKey),
				"i18n_key": i18nKey,
				"children": []map[string]any{},
			},
		}
		if parentID.Valid {
			parent := parentID.String
			node.parentID = &parent
		}
		nodes[menuID] = node
		order = append(order, node)
	}
	if err := menuRows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(order, func(i, j int) bool {
		if order[i].sortOrder == order[j].sortOrder {
			return order[i].menuID < order[j].menuID
		}
		return order[i].sortOrder < order[j].sortOrder
	})

	roots := make([]map[string]any, 0)
	for _, node := range order {
		if node.parentID == nil {
			roots = append(roots, node.payload)
			continue
		}
		parent, ok := nodes[*node.parentID]
		if !ok {
			roots = append(roots, node.payload)
			continue
		}
		children, _ := parent.payload["children"].([]map[string]any)
		parent.payload["children"] = append(children, node.payload)
	}

	permRows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT rp.perm_key
		FROM role_permissions rp
		JOIN workspace_members wm ON wm.role_id = rp.role_id
		WHERE wm.workspace_id = ? AND wm.user_id = ? AND wm.status = 'active'
		ORDER BY rp.perm_key`, workspaceID, user.UserID)
	if err != nil {
		return nil, err
	}
	defer permRows.Close()

	perms := make([]string, 0)
	for permRows.Next() {
		var permKey string
		if err := permRows.Scan(&permKey); err != nil {
			return nil, err
		}
		perms = append(perms, permKey)
	}
	if err := permRows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"workspace_id":  workspaceID,
		"menus":         roots,
		"permissions":   perms,
		"feature_flags": map[string]bool{},
	}, nil
}

// ValidateToken checks a raw token string and returns an AuthUser with all perms loaded.
func (s *AuthService) ValidateToken(token string) (*model.AuthUser, error) {
	tokenHash := hashToken(token)
	ctx := context.Background()

	var userID, email, displayName, expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.user_id, u.email, u.display_name, t.expires_at
		FROM auth_tokens t
		JOIN users u ON u.user_id = t.user_id
		WHERE t.token_hash = ? AND u.status = 'active'`, tokenHash).
		Scan(&userID, &email, &displayName, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}
	exp, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		return nil, fmt.Errorf("token expired")
	}
	user, err := s.loadAuthUserByIdentity(ctx, userID, email, displayName)
	if err != nil {
		return nil, err
	}

	// Update last_used_at
	_, _ = s.db.ExecContext(ctx,
		`UPDATE auth_tokens SET last_used_at=? WHERE token_hash=?`,
		time.Now().UTC().Format(time.RFC3339Nano), tokenHash)

	return user, nil
}

func (s *AuthService) loadAuthUserByID(ctx context.Context, userID string) (*model.AuthUser, error) {
	var email, displayName string
	err := s.db.QueryRowContext(ctx,
		`SELECT email, display_name FROM users WHERE user_id = ?`,
		userID,
	).Scan(&email, &displayName)
	if err != nil {
		return nil, err
	}
	return s.loadAuthUserByIdentity(ctx, userID, email, displayName)
}

func (s *AuthService) loadAuthUserByIdentity(
	ctx context.Context,
	userID, email, displayName string,
) (*model.AuthUser, error) {
	user := model.NewAuthUser(userID, email, displayName)

	memberRows, err := s.db.QueryContext(ctx, `
		SELECT workspace_id
		FROM workspace_members
		WHERE user_id = ? AND status = 'active'`, userID)
	if err != nil {
		return nil, err
	}
	for memberRows.Next() {
		var workspaceID string
		if err := memberRows.Scan(&workspaceID); err != nil {
			memberRows.Close()
			return nil, err
		}
		user.AddWorkspace(workspaceID)
	}
	if err := memberRows.Close(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT wm.workspace_id, rp.perm_key
		FROM workspace_members wm
		JOIN role_permissions rp ON rp.role_id = wm.role_id
		WHERE wm.user_id = ? AND wm.status = 'active'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var wsID, permKey string
		if err := rows.Scan(&wsID, &permKey); err != nil {
			return nil, err
		}
		user.GrantPerm(wsID, permKey)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueToken(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	tokenHash := hashToken(token)
	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenExpiry).Format(time.RFC3339Nano)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO auth_tokens(token_id,token_hash,user_id,expires_at,created_at) VALUES(?,?,?,?,?)`,
		uuid.NewString(), tokenHash, userID, expiresAt, now.Format(time.RFC3339Nano))
	if err != nil {
		return "", fmt.Errorf("issue token: %w", err)
	}
	return token, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

// RedactedDiagnostics is a placeholder used by the health handler.
func RedactedDiagnostics() map[string]any {
	return map[string]any{"status": "ok"}
}
