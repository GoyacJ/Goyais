package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db           *sql.DB
	tokenExpiry  time.Duration
}

func NewAuthService(db *sql.DB, tokenExpiryHours int) *AuthService {
	return &AuthService{
		db:          db,
		tokenExpiry: time.Duration(tokenExpiryHours) * time.Hour,
	}
}

func (s *AuthService) IsBootstrapComplete(ctx context.Context) (bool, error) {
	var done int
	err := s.db.QueryRowContext(ctx,
		`SELECT setup_completed FROM system_state WHERE singleton_id = 1`).Scan(&done)
	if err != nil {
		return false, err
	}
	return done == 1, nil
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
	return map[string]any{
		"user_id":      user.UserID,
		"email":        user.Email,
		"display_name": user.DisplayName,
	}, nil
}

func (s *AuthService) Navigation(ctx context.Context, workspaceID string) ([]map[string]any, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.menu_id, m.parent_id, m.sort_order, m.route, m.icon_key, m.i18n_key
		FROM menus m
		JOIN role_menus rm ON rm.menu_id = m.menu_id
		JOIN workspace_members wm ON wm.role_id = rm.role_id
		WHERE wm.workspace_id = ? AND wm.user_id = ? AND wm.status = 'active'
		ORDER BY m.sort_order`, workspaceID, user.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nav []map[string]any
	for rows.Next() {
		var menuID, route, iconKey, i18nKey string
		var parentID sql.NullString
		var sortOrder int
		if err := rows.Scan(&menuID, &parentID, &sortOrder, &route, &iconKey, &i18nKey); err != nil {
			return nil, err
		}
		item := map[string]any{
			"menu_id":    menuID,
			"sort_order": sortOrder,
			"route":      route,
			"icon_key":   iconKey,
			"i18n_key":   i18nKey,
		}
		if parentID.Valid {
			item["parent_id"] = parentID.String
		}
		nav = append(nav, item)
	}
	return nav, nil
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

	user := model.NewAuthUser(userID, email, displayName)

	// Load all workspace permissions for this user
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

	// Update last_used_at
	_, _ = s.db.ExecContext(ctx,
		`UPDATE auth_tokens SET last_used_at=? WHERE token_hash=?`,
		time.Now().UTC().Format(time.RFC3339Nano), tokenHash)

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

// RedactedDiagnostics is a placeholder used by the health handler.
func RedactedDiagnostics() map[string]any {
	return map[string]any{"status": "ok"}
}
