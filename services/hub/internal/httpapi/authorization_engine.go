package httpapi

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
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
	"conversation.read":        {PermissionKey: "conversation.read", RiskLevel: "low", OperationType: "read"},
	"conversation.write":       {PermissionKey: "conversation.write", RiskLevel: "medium", OperationType: "write", ABACRequired: true},
	"execution.control":        {PermissionKey: "execution.control", RiskLevel: "high", OperationType: "execute", ABACRequired: true},
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
	session, err := resolveSessionForWorkspace(state, r, workspaceID)
	if err != nil {
		appendAuthzAudit(state, r, workspaceID, "", action, resource, "denied", map[string]any{"reason": err.code})
		return Session{}, err
	}

	normalizedWorkspace := strings.TrimSpace(workspaceID)
	if normalizedWorkspace == "" {
		normalizedWorkspace = strings.TrimSpace(session.WorkspaceID)
	}
	if normalizedWorkspace != "" && session.WorkspaceID != localWorkspaceID && session.WorkspaceID != normalizedWorkspace {
		appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "workspace_mismatch"})
		return Session{}, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Workspace access is denied",
			details: map[string]any{"workspace_id": normalizedWorkspace},
		}
	}
	if len(allowedRoles) > 0 && !slices.Contains(allowedRoles, session.Role) {
		appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "role_forbidden"})
		return Session{}, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Permission is denied",
			details: map[string]any{"required_roles": allowedRoles},
		}
	}
	if session.WorkspaceID == localWorkspaceID {
		targetWorkspace := resolveTargetWorkspace(session, normalizedWorkspace, resource)
		if targetWorkspace != "" && targetWorkspace != localWorkspaceID {
			appendAuthzAudit(state, r, targetWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "workspace_mismatch"})
			return Session{}, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Workspace access is denied",
				details: map[string]any{"workspace_id": targetWorkspace},
			}
		}
		appendAuthzAudit(state, r, localWorkspaceID, session.UserID, action, resource, "success", map[string]any{"mode": "local"})
		return session, nil
	}

	spec := actionSpecs[action]
	permissionKey := spec.PermissionKey
	if strings.TrimSpace(permissionKey) == "" {
		permissionKey = action
	}
	if state.authz != nil {
		permissions, loadErr := state.authz.listRolePermissions(normalizedWorkspace, session.Role)
		if loadErr != nil {
			return Session{}, &apiError{
				status:  http.StatusInternalServerError,
				code:    "AUTHZ_INTERNAL_ERROR",
				message: "Failed to load role permissions",
				details: map[string]any{},
			}
		}
		if !containsPermission(permissions, permissionKey) {
			appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "rbac_forbidden", "permission": permissionKey})
			return Session{}, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Permission is denied",
				details: map[string]any{"permission": permissionKey},
			}
		}
	}

	ctx := authorizationContext{
		RiskLevel:     firstNonEmpty(input.RiskLevel, spec.RiskLevel, "low"),
		OperationType: firstNonEmpty(input.OperationType, spec.OperationType, "read"),
		RequestSource: firstNonEmpty(input.RequestSource, "api"),
		ABACRequired:  input.ABACRequired || spec.ABACRequired || input.OperationType == "write",
	}
	resource.WorkspaceID = firstNonEmpty(resource.WorkspaceID, normalizedWorkspace)

	if state.authz != nil && ctx.ABACRequired {
		policies, loadErr := state.authz.listABACPolicies(normalizedWorkspace)
		if loadErr != nil {
			return Session{}, &apiError{
				status:  http.StatusInternalServerError,
				code:    "AUTHZ_INTERNAL_ERROR",
				message: "Failed to load ABAC policies",
				details: map[string]any{},
			}
		}
		allowMatched := false
		for _, policy := range policies {
			if !policy.Enabled {
				continue
			}
			if !matchABACPolicy(policy, session, resource, action, ctx) {
				continue
			}
			if policy.Effect == ABACEffectDeny {
				appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "abac_deny", "policy_id": policy.ID})
				return Session{}, &apiError{
					status:  http.StatusForbidden,
					code:    "ACCESS_DENIED",
					message: "Permission is denied by ABAC policy",
					details: map[string]any{"policy_id": policy.ID},
				}
			}
			allowMatched = true
		}
		if !allowMatched {
			appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "denied", map[string]any{"reason": "abac_no_allow"})
			return Session{}, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Permission is denied by ABAC policy",
				details: map[string]any{"action": action},
			}
		}
	}

	appendAuthzAudit(state, r, normalizedWorkspace, session.UserID, action, resource, "success", map[string]any{
		"risk_level":     ctx.RiskLevel,
		"operation_type": ctx.OperationType,
	})
	return session, nil
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
