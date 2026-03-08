package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	unifiedpolicy "goyais/services/hub/internal/agent/policy/unified"
)

type actionSpec struct {
	PermissionKey string
	RiskLevel     string
	OperationType string
	ABACRequired  bool
}

type authorizationResource struct {
	WorkspaceID  string
	OwnerUserID  string
	Scope        string
	ResourceType string
	ShareStatus  string
	TargetID     string
}

type authorizationContext struct {
	RiskLevel     string
	OperationType string
	RequestSource string
	ABACRequired  bool
}

var actionSpecs = map[string]actionSpec{
	"project.read":             {PermissionKey: "project.read", RiskLevel: "low", OperationType: "read"},
	"project.write":            {PermissionKey: "project.write", RiskLevel: "medium", OperationType: "write", ABACRequired: true},
	"session.read":             {PermissionKey: "session.read", RiskLevel: "low", OperationType: "read"},
	"session.write":            {PermissionKey: "session.write", RiskLevel: "medium", OperationType: "write", ABACRequired: true},
	"run.control":              {PermissionKey: "run.control", RiskLevel: "high", OperationType: "execute", ABACRequired: true},
	"resource.read":            {PermissionKey: "resource.read", RiskLevel: "low", OperationType: "read"},
	"resource.write":           {PermissionKey: "resource.write", RiskLevel: "medium", OperationType: "write", ABACRequired: true},
	"resource_config.read":     {PermissionKey: "resource_config.read", RiskLevel: "low", OperationType: "read"},
	"resource_config.write":    {PermissionKey: "resource_config.write", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"resource_config.delete":   {PermissionKey: "resource_config.delete", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"project_config.read":      {PermissionKey: "project_config.read", RiskLevel: "low", OperationType: "read"},
	"catalog.update_root":      {PermissionKey: "catalog.update_root", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"model.test":               {PermissionKey: "model.test", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"mcp.connect":              {PermissionKey: "mcp.connect", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"share.request":            {PermissionKey: "share.request", RiskLevel: "medium", OperationType: "write", ABACRequired: true},
	"share.approve":            {PermissionKey: "share.approve", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"share.reject":             {PermissionKey: "share.reject", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"share.revoke":             {PermissionKey: "share.revoke", RiskLevel: "high", OperationType: "write", ABACRequired: true},
	"admin.users.manage":       {PermissionKey: "admin.users.manage", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"admin.roles.manage":       {PermissionKey: "admin.roles.manage", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"admin.permissions.manage": {PermissionKey: "admin.permissions.manage", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"admin.menus.manage":       {PermissionKey: "admin.menus.manage", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"admin.policies.manage":    {PermissionKey: "admin.policies.manage", RiskLevel: "critical", OperationType: "write", ABACRequired: true},
	"admin.audit.read":         {PermissionKey: "admin.audit.read", RiskLevel: "low", OperationType: "read"},
}

func authorizeAction(state *AppState, r *http.Request, workspaceID string, action string, resource authorizationResource, input authorizationContext, allowedRoles ...Role) (Session, *apiError) {
	if state == nil {
		return Session{}, &apiError{
			status:  500,
			code:    "AUTHZ_INTERNAL_ERROR",
			message: "Authorization state is not configured",
			details: map[string]any{},
		}
	}
	gate := state.unifiedPermissionGate
	if gate == nil {
		gate = newUnifiedAuthorizationGate(state)
	}
	if gate == nil {
		return Session{}, &apiError{
			status:  500,
			code:    "AUTHZ_INTERNAL_ERROR",
			message: "Unified permission gate is not configured",
			details: map[string]any{},
		}
	}

	roles := make([]string, 0, len(allowedRoles))
	for _, role := range allowedRoles {
		roles = append(roles, string(role))
	}
	decision, err := gate.Authorize(r.Context(), unifiedpolicy.Request{
		Action:       action,
		WorkspaceID:  strings.TrimSpace(workspaceID),
		AccessToken:  extractAccessToken(r),
		TraceID:      TraceIDFromContext(r.Context()),
		AllowedRoles: roles,
		Resource: unifiedpolicy.Resource{
			WorkspaceID:  strings.TrimSpace(resource.WorkspaceID),
			OwnerUserID:  strings.TrimSpace(resource.OwnerUserID),
			Scope:        strings.TrimSpace(resource.Scope),
			ResourceType: strings.TrimSpace(resource.ResourceType),
			ShareStatus:  strings.TrimSpace(resource.ShareStatus),
			TargetID:     firstNonEmpty(strings.TrimSpace(resource.TargetID), strings.TrimSpace(resource.OwnerUserID), strings.TrimSpace(resource.WorkspaceID), "unknown"),
		},
		Context: unifiedpolicy.Context{
			RiskLevel:     strings.TrimSpace(input.RiskLevel),
			OperationType: strings.TrimSpace(input.OperationType),
			RequestSource: strings.TrimSpace(input.RequestSource),
			ABACRequired:  input.ABACRequired,
		},
	})
	if err != nil {
		var authErr *apiError
		if errors.As(err, &authErr) {
			return Session{}, authErr
		}
		return Session{}, &apiError{
			status:  500,
			code:    "AUTHZ_INTERNAL_ERROR",
			message: "Authorization evaluation failed",
			details: map[string]any{"error": err.Error()},
		}
	}
	if !decision.Allowed {
		return Session{}, &apiError{
			status:  decision.StatusCode,
			code:    decision.Code,
			message: decision.Message,
			details: cloneMapAny(decision.Details),
		}
	}
	return sessionFromSubject(decision.Subject), nil
}

func matchABACPolicy(policy ABACPolicy, session Session, resource authorizationResource, action string, input authorizationContext) bool {
	env := map[string]map[string]any{
		"subject": {
			"user_id":      session.UserID,
			"roles":        []string{string(session.Role)},
			"workspace_id": session.WorkspaceID,
		},
		"resource": {
			"workspace_id":  resource.WorkspaceID,
			"owner_user_id": resource.OwnerUserID,
			"scope":         resource.Scope,
			"resource_type": resource.ResourceType,
			"share_status":  resource.ShareStatus,
		},
		"action": {
			"name": action,
		},
		"context": {
			"risk_level":     input.RiskLevel,
			"operation_type": input.OperationType,
			"request_source": input.RequestSource,
		},
	}
	return evaluateExpression(policy.SubjectExpr, env["subject"], env) &&
		evaluateExpression(policy.ResourceExpr, env["resource"], env) &&
		evaluateExpression(policy.ActionExpr, env["action"], env) &&
		evaluateExpression(policy.ContextExpr, env["context"], env)
}

func evaluateExpression(expr map[string]any, scope map[string]any, env map[string]map[string]any) bool {
	if len(expr) == 0 {
		return true
	}
	for field, rawRule := range expr {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			return false
		}
		actual := scope[field]
		for operator, expected := range rule {
			resolved := resolveExpressionValue(expected, env)
			if !matchOperator(operator, actual, resolved) {
				return false
			}
		}
	}
	return true
}

func resolveExpressionValue(value any, env map[string]map[string]any) any {
	switch typed := value.(type) {
	case string:
		if !strings.HasPrefix(typed, "$") {
			return typed
		}
		parts := strings.Split(strings.TrimPrefix(typed, "$"), ".")
		if len(parts) != 2 {
			return typed
		}
		group := env[parts[0]]
		if group == nil {
			return typed
		}
		resolved, exists := group[parts[1]]
		if !exists {
			return typed
		}
		return resolved
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, resolveExpressionValue(item, env))
		}
		return items
	default:
		return value
	}
}

func matchOperator(operator string, actual any, expected any) bool {
	switch operator {
	case "eq":
		return compareString(actual) == compareString(expected)
	case "neq":
		return compareString(actual) != compareString(expected)
	case "in":
		values := toAnySlice(expected)
		actualSlice := toAnySlice(actual)
		if len(actualSlice) > 0 {
			for _, actualItem := range actualSlice {
				actualString := compareString(actualItem)
				for _, item := range values {
					if compareString(item) == actualString {
						return true
					}
				}
			}
			return false
		}
		actualString := compareString(actual)
		for _, item := range values {
			if compareString(item) == actualString {
				return true
			}
		}
		return false
	case "contains":
		actualSlice := toAnySlice(actual)
		expectedValue := compareString(expected)
		for _, item := range actualSlice {
			if compareString(item) == expectedValue {
				return true
			}
		}
		if text := compareString(actual); text != "" {
			return strings.Contains(text, expectedValue)
		}
		return false
	default:
		return false
	}
}

func compareString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func toAnySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	default:
		return []any{}
	}
}

func containsPermission(permissions []string, target string) bool {
	if target == "" {
		return true
	}
	for _, permission := range permissions {
		if permission == "*" || permission == target {
			return true
		}
	}
	return false
}

func appendAuthzAudit(state *AppState, r *http.Request, workspaceID string, actorUserID string, action string, resource authorizationResource, result string, details map[string]any) {
	if state == nil {
		return
	}
	resourceID := firstNonEmpty(resource.OwnerUserID, resource.WorkspaceID, "unknown")
	state.AppendAudit(AdminAuditEvent{
		Actor:    firstNonEmpty(actorUserID, "anonymous"),
		Action:   "authz." + action,
		Resource: resourceID,
		Result:   result,
		TraceID:  TraceIDFromContext(r.Context()),
	})
	if state.authz != nil {
		_ = state.authz.appendAudit(
			firstNonEmpty(workspaceID, localWorkspaceID),
			actorUserID,
			"authz."+action,
			"resource",
			resourceID,
			result,
			details,
			TraceIDFromContext(r.Context()),
		)
	}
}
