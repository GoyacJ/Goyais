package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
	"goyais/internal/config"
)

var (
	contextModeMu sync.RWMutex
	contextMode   = config.AuthContextModeJWTOrHeader
)

type requestContextError struct {
	Status     int
	Code       string
	MessageKey string
	Details    map[string]any
}

func (e *requestContextError) Error() string {
	return e.Code
}

type jwtContextClaims struct {
	TenantID          string
	WorkspaceID       string
	UserID            string
	Roles             []string
	AllowedTenantIDs  map[string]struct{}
	AllowedWorkspaceIDs map[string]struct{}
	AllowedRoles      map[string]struct{}
}

func configureAuthContext(cfg config.Config) {
	contextModeMu.Lock()
	defer contextModeMu.Unlock()
	contextMode = normalizeContextMode(cfg.Authz.ContextMode)
}

func normalizeContextMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case config.AuthContextModeHeaderOnly:
		return config.AuthContextModeHeaderOnly
	case config.AuthContextModeJWTOrHeader:
		return config.AuthContextModeJWTOrHeader
	default:
		return config.AuthContextModeJWTOrHeader
	}
}

func currentContextMode() string {
	contextModeMu.RLock()
	defer contextModeMu.RUnlock()
	return contextMode
}

func requireRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	return extractRequestContext(w, r)
}

func extractRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	if currentContextMode() == config.AuthContextModeHeaderOnly {
		return extractHeaderRequestContext(w, r)
	}
	return extractJWTOrHeaderRequestContext(w, r)
}

func extractJWTOrHeaderRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	rawAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	if rawAuth == "" {
		return extractHeaderRequestContext(w, r)
	}

	token, err := parseBearerToken(rawAuth)
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_TOKEN", "error.context.invalid_token", map[string]any{"reason": "invalid_bearer"})
		return command.RequestContext{}, false
	}

	claims, err := parseJWTContextClaims(token)
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_TOKEN", "error.context.invalid_token", map[string]any{"reason": "invalid_claims"})
		return command.RequestContext{}, false
	}

	reqCtx, err := resolveContextFromJWTClaims(claims, r.Header)
	if err != nil {
		var contextErr *requestContextError
		if errors.As(err, &contextErr) {
			errorx.Write(w, contextErr.Status, contextErr.Code, contextErr.MessageKey, contextErr.Details)
			return command.RequestContext{}, false
		}
		errorx.Write(w, http.StatusBadRequest, "INVALID_TOKEN", "error.context.invalid_token", map[string]any{"reason": "invalid_context"})
		return command.RequestContext{}, false
	}

	return reqCtx, true
}

func extractHeaderRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
	workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	roles := parseRolesHeader(r.Header.Get("X-Roles"))
	policyVersion := strings.TrimSpace(r.Header.Get("X-Policy-Version"))
	if policyVersion == "" {
		policyVersion = "v0.1"
	}
	traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
	if traceID == "" {
		traceID = newTraceID()
	}

	missing := make([]string, 0, 3)
	if tenantID == "" {
		missing = append(missing, "X-Tenant-Id")
	}
	if workspaceID == "" {
		missing = append(missing, "X-Workspace-Id")
	}
	if userID == "" {
		missing = append(missing, "X-User-Id")
	}

	if len(missing) > 0 {
		errorx.Write(w, http.StatusBadRequest, "MISSING_CONTEXT", "error.context.missing", map[string]any{
			"missingHeaders": missing,
		})
		return command.RequestContext{}, false
	}

	return command.RequestContext{
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		UserID:        userID,
		OwnerID:       userID,
		Roles:         roles,
		PolicyVersion: policyVersion,
		TraceID:       traceID,
	}, true
}

func resolveContextFromJWTClaims(claims jwtContextClaims, headers http.Header) (command.RequestContext, error) {
	tenantID := claims.TenantID
	if v := strings.TrimSpace(headers.Get("X-Tenant-Id")); v != "" {
		if _, ok := claims.AllowedTenantIDs[v]; !ok {
			return command.RequestContext{}, &requestContextError{
				Status:     http.StatusForbidden,
				Code:       "FORBIDDEN",
				MessageKey: "error.authz.forbidden",
				Details:    map[string]any{"reason": "tenant_out_of_scope"},
			}
		}
		tenantID = v
	}

	workspaceID := claims.WorkspaceID
	if v := strings.TrimSpace(headers.Get("X-Workspace-Id")); v != "" {
		if _, ok := claims.AllowedWorkspaceIDs[v]; !ok {
			return command.RequestContext{}, &requestContextError{
				Status:     http.StatusForbidden,
				Code:       "FORBIDDEN",
				MessageKey: "error.authz.forbidden",
				Details:    map[string]any{"reason": "workspace_out_of_scope"},
			}
		}
		workspaceID = v
	}

	userID := claims.UserID
	if v := strings.TrimSpace(headers.Get("X-User-Id")); v != "" && v != claims.UserID {
		return command.RequestContext{}, &requestContextError{
			Status:     http.StatusForbidden,
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Details:    map[string]any{"reason": "user_out_of_scope"},
		}
	}

	roles := claims.Roles
	rawRoles := strings.TrimSpace(headers.Get("X-Roles"))
	if rawRoles != "" {
		headerRoles := parseRolesCSV(rawRoles, true)
		if len(headerRoles) == 0 {
			return command.RequestContext{}, &requestContextError{
				Status:     http.StatusForbidden,
				Code:       "FORBIDDEN",
				MessageKey: "error.authz.forbidden",
				Details:    map[string]any{"reason": "roles_out_of_scope"},
			}
		}
		for _, role := range headerRoles {
			if _, ok := claims.AllowedRoles[role]; !ok {
				return command.RequestContext{}, &requestContextError{
					Status:     http.StatusForbidden,
					Code:       "FORBIDDEN",
					MessageKey: "error.authz.forbidden",
					Details:    map[string]any{"reason": "roles_out_of_scope"},
				}
			}
		}
		roles = headerRoles
	}

	policyVersion := strings.TrimSpace(headers.Get("X-Policy-Version"))
	if policyVersion == "" {
		policyVersion = "v0.1"
	}
	traceID := strings.TrimSpace(headers.Get("X-Trace-Id"))
	if traceID == "" {
		traceID = newTraceID()
	}

	return command.RequestContext{
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		UserID:        userID,
		OwnerID:       userID,
		Roles:         roles,
		PolicyVersion: policyVersion,
		TraceID:       traceID,
	}, nil
}

func parseBearerToken(rawAuth string) (string, error) {
	const prefix = "bearer "
	if len(rawAuth) < len(prefix) || !strings.EqualFold(rawAuth[:len(prefix)], prefix) {
		return "", errors.New("authorization header is not bearer")
	}
	token := strings.TrimSpace(rawAuth[len(prefix):])
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

func parseJWTContextClaims(token string) (jwtContextClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return jwtContextClaims{}, errors.New("token has invalid format")
	}

	payloadRaw, err := decodeJWTPart(parts[1])
	if err != nil {
		return jwtContextClaims{}, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return jwtContextClaims{}, err
	}

	tenantID := claimString(payload, "tenantId", "tenant_id")
	workspaceID := claimString(payload, "workspaceId", "workspace_id")
	userID := claimString(payload, "userId", "user_id", "sub")
	if tenantID == "" || workspaceID == "" || userID == "" {
		return jwtContextClaims{}, errors.New("required claims are missing")
	}

	roles := claimStringSlice(payload, true, "roles", "role")
	if len(roles) == 0 {
		roles = []string{"member"}
	}
	allowedTenantIDs := claimStringSlice(payload, false, "tenantIds", "tenant_ids", "allowedTenantIds", "allowed_tenant_ids")
	if len(allowedTenantIDs) == 0 {
		allowedTenantIDs = []string{tenantID}
	}
	allowedWorkspaceIDs := claimStringSlice(payload, false, "workspaceIds", "workspace_ids", "allowedWorkspaceIds", "allowed_workspace_ids")
	if len(allowedWorkspaceIDs) == 0 {
		allowedWorkspaceIDs = []string{workspaceID}
	}

	return jwtContextClaims{
		TenantID:            tenantID,
		WorkspaceID:         workspaceID,
		UserID:              userID,
		Roles:               roles,
		AllowedTenantIDs:    toSet(allowedTenantIDs),
		AllowedWorkspaceIDs: toSet(allowedWorkspaceIDs),
		AllowedRoles:        toSet(roles),
	}, nil
}

func decodeJWTPart(raw string) ([]byte, error) {
	out, err := base64.RawURLEncoding.DecodeString(raw)
	if err == nil {
		return out, nil
	}
	// Fallback for tokens that include standard base64 padding.
	return base64.URLEncoding.DecodeString(raw)
}

func claimString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		str, ok := value.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)
		if str != "" {
			return str
		}
	}
	return ""
}

func claimStringSlice(payload map[string]any, lowercase bool, keys ...string) []string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		items := parseClaimList(raw, lowercase)
		if len(items) > 0 {
			return items
		}
	}
	return nil
}

func parseClaimList(raw any, lowercase bool) []string {
	switch value := raw.(type) {
	case string:
		if strings.Contains(value, ",") {
			return parseRolesCSV(value, lowercase)
		}
		return parseRolesCSV(value, lowercase)
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			str, ok := item.(string)
			if !ok {
				continue
			}
			parts = append(parts, str)
		}
		return parseRolesCSV(strings.Join(parts, ","), lowercase)
	default:
		return nil
	}
}

func parseRolesHeader(raw string) []string {
	roles := parseRolesCSV(raw, true)
	if len(roles) == 0 {
		return []string{"member"}
	}
	return roles
}

func parseRolesCSV(raw string, lowercase bool) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if lowercase {
			value = strings.ToLower(value)
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func newTraceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "trace_generated"
	}
	return "trace_" + hex.EncodeToString(buf)
}
