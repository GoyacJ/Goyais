package httpapi

import (
	"context"
	"net/http"
	"slices"
	"strings"

	unifiedpolicy "goyais/services/hub/internal/agent/policy/unified"
)

type authorizationGate interface {
	Authorize(ctx context.Context, req unifiedpolicy.Request) (unifiedpolicy.Decision, error)
	RecordOperation(ctx context.Context, event unifiedpolicy.AuditEvent) error
}

type authorizationEvaluator struct {
	state *AppState
}

type authorizationHookObserver struct {
	state *AppState
}

func newUnifiedAuthorizationGate(state *AppState) authorizationGate {
	if state == nil {
		return nil
	}
	return unifiedpolicy.NewGate(
		authorizationEvaluator{state: state},
		unifiedpolicy.Options{
			AuditLogger:  newAuthorizationAuditLogger(state),
			HookObserver: authorizationHookObserver{state: state},
		},
	)
}

func (e authorizationEvaluator) Evaluate(_ context.Context, req unifiedpolicy.Request) (unifiedpolicy.Decision, error) {
	session, sessionErr := resolveSessionForAccessToken(e.state, req.AccessToken, req.WorkspaceID)
	if sessionErr != nil {
		return buildDeniedAuthorizationDecision(req, Session{}, sessionErr, map[string]any{
			"reason": sessionErr.code,
		}), nil
	}
	subject := subjectFromSession(session)

	normalizedWorkspace := strings.TrimSpace(req.WorkspaceID)
	if normalizedWorkspace == "" {
		normalizedWorkspace = strings.TrimSpace(session.WorkspaceID)
	}
	if normalizedWorkspace != "" && session.WorkspaceID != localWorkspaceID && session.WorkspaceID != normalizedWorkspace {
		return buildDeniedAuthorizationDecision(req, session, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Workspace access is denied",
			details: map[string]any{"workspace_id": normalizedWorkspace},
		}, map[string]any{
			"reason": "workspace_mismatch",
		}), nil
	}
	if len(req.AllowedRoles) > 0 && !slices.Contains(req.AllowedRoles, string(session.Role)) {
		return buildDeniedAuthorizationDecision(req, session, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Permission is denied",
			details: map[string]any{"required_roles": req.AllowedRoles},
		}, map[string]any{
			"reason": "role_forbidden",
		}), nil
	}

	if session.WorkspaceID == localWorkspaceID {
		targetWorkspace := resolveTargetWorkspace(session, normalizedWorkspace, authorizationResource{
			WorkspaceID:  req.Resource.WorkspaceID,
			OwnerUserID:  req.Resource.OwnerUserID,
			Scope:        req.Resource.Scope,
			ResourceType: req.Resource.ResourceType,
			ShareStatus:  req.Resource.ShareStatus,
		})
		if targetWorkspace != "" && targetWorkspace != localWorkspaceID {
			return buildDeniedAuthorizationDecision(req, session, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Workspace access is denied",
				details: map[string]any{"workspace_id": targetWorkspace},
			}, map[string]any{
				"reason": "workspace_mismatch",
			}), nil
		}
		return unifiedpolicy.Decision{
			Allowed: true,
			Result:  "success",
			Reason:  "local workspace access granted",
			Subject: subject,
			AuditEvent: unifiedpolicy.AuditEvent{
				WorkspaceID: localWorkspaceID,
				ActorID:     firstNonEmpty(session.UserID, "anonymous"),
				Action:      "authz." + req.Action,
				TargetType:  "resource",
				TargetID:    authorizationAuditTargetID(req.Resource),
				Result:      "success",
				TraceID:     req.TraceID,
				Details:     map[string]any{"mode": "local"},
			},
		}, nil
	}

	spec := actionSpecs[req.Action]
	permissionKey := spec.PermissionKey
	if strings.TrimSpace(permissionKey) == "" {
		permissionKey = req.Action
	}
	if e.state.authz != nil {
		permissions, loadErr := e.state.authz.listRolePermissions(normalizedWorkspace, session.Role)
		if loadErr != nil {
			return unifiedpolicy.Decision{}, &apiError{
				status:  http.StatusInternalServerError,
				code:    "AUTHZ_INTERNAL_ERROR",
				message: "Failed to load role permissions",
				details: map[string]any{},
			}
		}
		if !containsPermission(permissions, permissionKey) {
			return buildDeniedAuthorizationDecision(req, session, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Permission is denied",
				details: map[string]any{"permission": permissionKey},
			}, map[string]any{
				"reason":     "rbac_forbidden",
				"permission": permissionKey,
			}), nil
		}
	}

	ctx := authorizationContext{
		RiskLevel:     firstNonEmpty(req.Context.RiskLevel, spec.RiskLevel, "low"),
		OperationType: firstNonEmpty(req.Context.OperationType, spec.OperationType, "read"),
		RequestSource: firstNonEmpty(req.Context.RequestSource, "api"),
		ABACRequired:  req.Context.ABACRequired || spec.ABACRequired || req.Context.OperationType == "write",
	}
	resource := authorizationResource{
		WorkspaceID:  firstNonEmpty(req.Resource.WorkspaceID, normalizedWorkspace),
		OwnerUserID:  req.Resource.OwnerUserID,
		Scope:        req.Resource.Scope,
		ResourceType: req.Resource.ResourceType,
		ShareStatus:  req.Resource.ShareStatus,
	}

	if e.state.authz != nil && ctx.ABACRequired {
		policies, loadErr := e.state.authz.listABACPolicies(normalizedWorkspace)
		if loadErr != nil {
			return unifiedpolicy.Decision{}, &apiError{
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
			if !matchABACPolicy(policy, session, resource, req.Action, ctx) {
				continue
			}
			if policy.Effect == ABACEffectDeny {
				return buildDeniedAuthorizationDecision(req, session, &apiError{
					status:  http.StatusForbidden,
					code:    "ACCESS_DENIED",
					message: "Permission is denied by ABAC policy",
					details: map[string]any{"policy_id": policy.ID},
				}, map[string]any{
					"reason":    "abac_deny",
					"policy_id": policy.ID,
				}), nil
			}
			allowMatched = true
		}
		if !allowMatched {
			return buildDeniedAuthorizationDecision(req, session, &apiError{
				status:  http.StatusForbidden,
				code:    "ACCESS_DENIED",
				message: "Permission is denied by ABAC policy",
				details: map[string]any{"action": req.Action},
			}, map[string]any{
				"reason": "abac_no_allow",
			}), nil
		}
	}

	return unifiedpolicy.Decision{
		Allowed: true,
		Result:  "success",
		Reason:  "authorization granted",
		Subject: subject,
		AuditEvent: unifiedpolicy.AuditEvent{
			WorkspaceID: normalizedWorkspace,
			ActorID:     firstNonEmpty(session.UserID, "anonymous"),
			Action:      "authz." + req.Action,
			TargetType:  "resource",
			TargetID:    authorizationAuditTargetID(req.Resource),
			Result:      "success",
			TraceID:     req.TraceID,
			Details: map[string]any{
				"risk_level":     ctx.RiskLevel,
				"operation_type": ctx.OperationType,
			},
		},
	}, nil
}

func newAuthorizationAuditLogger(state *AppState) unifiedpolicy.AuditLogger {
	return unifiedpolicy.AuditLoggerFunc(func(_ context.Context, event unifiedpolicy.AuditEvent) error {
		return recordAuthorizationAudit(state, event)
	})
}

func recordAuthorizationAudit(state *AppState, event unifiedpolicy.AuditEvent) error {
	if state == nil {
		return nil
	}
	state.AppendAudit(AdminAuditEvent{
		Actor:    firstNonEmpty(event.ActorID, "anonymous"),
		Action:   strings.TrimSpace(event.Action),
		Resource: firstNonEmpty(event.TargetID, "unknown"),
		Result:   strings.TrimSpace(event.Result),
		TraceID:  strings.TrimSpace(event.TraceID),
	})
	if state.authz != nil {
		_ = state.authz.appendAudit(
			firstNonEmpty(event.WorkspaceID, localWorkspaceID),
			event.ActorID,
			event.Action,
			firstNonEmpty(event.TargetType, "resource"),
			firstNonEmpty(event.TargetID, "unknown"),
			event.Result,
			event.Details,
			event.TraceID,
		)
	}
	return nil
}

func (o authorizationHookObserver) Observe(ctx context.Context, req unifiedpolicy.Request, decision unifiedpolicy.Decision) error {
	if o.state == nil {
		return nil
	}
	scope := resolveAuthorizationHookScope(o.state, req, decision)
	hookDecision, matchedPolicyID := evaluateHookDecisionForContextWithState(
		o.state,
		hookEvaluationContext{
			WorkspaceID:      scope.WorkspaceID,
			ProjectID:        scope.ProjectID,
			SessionID:        scope.SessionID,
			ToolName:         strings.TrimSpace(req.Action),
			IsLocalWorkspace: strings.TrimSpace(scope.WorkspaceID) == localWorkspaceID,
		},
		HookEventTypePermissionRequest,
	)

	details := map[string]any{
		"authz_action":   strings.TrimSpace(req.Action),
		"authz_result":   firstNonEmpty(strings.TrimSpace(decision.Result), strings.TrimSpace(decision.AuditEvent.Result), "success"),
		"operation_type": strings.TrimSpace(req.Context.OperationType),
		"request_source": firstNonEmpty(strings.TrimSpace(req.Context.RequestSource), "api"),
	}
	if strings.TrimSpace(scope.ProjectID) != "" {
		details["project_id"] = strings.TrimSpace(scope.ProjectID)
	}
	if strings.TrimSpace(scope.SessionID) != "" {
		details["session_id"] = strings.TrimSpace(scope.SessionID)
	}
	if strings.TrimSpace(matchedPolicyID) != "" {
		details["policy_id"] = strings.TrimSpace(matchedPolicyID)
	}
	if strings.TrimSpace(hookDecision.Reason) != "" {
		details["reason"] = strings.TrimSpace(hookDecision.Reason)
	}

	targetType := "resource"
	if strings.TrimSpace(scope.SessionID) != "" {
		targetType = "session"
	} else if strings.TrimSpace(scope.ProjectID) != "" {
		targetType = "project"
	}

	return recordAuthorizationAudit(o.state, unifiedpolicy.AuditEvent{
		WorkspaceID: firstNonEmpty(strings.TrimSpace(scope.WorkspaceID), localWorkspaceID),
		ActorID: firstNonEmpty(
			strings.TrimSpace(decision.Subject.ID),
			strings.TrimSpace(req.Subject.ID),
			"anonymous",
		),
		Action:     "hook.permission_request",
		TargetType: targetType,
		TargetID: firstNonEmpty(
			strings.TrimSpace(scope.TargetID),
			authorizationAuditTargetID(req.Resource),
		),
		Result:  string(hookDecision.Action),
		TraceID: firstNonEmpty(strings.TrimSpace(req.TraceID), strings.TrimSpace(decision.AuditEvent.TraceID)),
		Details: details,
	})
}

type authorizationHookScope struct {
	WorkspaceID string
	ProjectID   string
	SessionID   string
	TargetID    string
}

func resolveAuthorizationHookScope(state *AppState, req unifiedpolicy.Request, decision unifiedpolicy.Decision) authorizationHookScope {
	scope := authorizationHookScope{
		WorkspaceID: firstNonEmpty(
			strings.TrimSpace(req.WorkspaceID),
			strings.TrimSpace(req.Resource.WorkspaceID),
			strings.TrimSpace(decision.Subject.WorkspaceID),
			localWorkspaceID,
		),
		TargetID: authorizationAuditTargetID(req.Resource),
	}
	if state == nil {
		return scope
	}

	targetID := strings.TrimSpace(req.Resource.TargetID)
	if targetID == "" {
		return scope
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	if execution, ok := state.executions[targetID]; ok {
		scope.TargetID = execution.ID
		scope.SessionID = strings.TrimSpace(execution.ConversationID)
		scope.WorkspaceID = firstNonEmpty(scope.WorkspaceID, strings.TrimSpace(execution.WorkspaceID))
		if conversation, exists := state.conversations[execution.ConversationID]; exists {
			scope.ProjectID = strings.TrimSpace(conversation.ProjectID)
			scope.WorkspaceID = firstNonEmpty(scope.WorkspaceID, strings.TrimSpace(conversation.WorkspaceID))
		}
		return scope
	}
	if conversation, ok := state.conversations[targetID]; ok {
		scope.TargetID = conversation.ID
		scope.SessionID = conversation.ID
		scope.ProjectID = strings.TrimSpace(conversation.ProjectID)
		scope.WorkspaceID = firstNonEmpty(scope.WorkspaceID, strings.TrimSpace(conversation.WorkspaceID))
		return scope
	}
	if project, ok := state.projects[targetID]; ok {
		scope.TargetID = project.ID
		scope.ProjectID = project.ID
		scope.WorkspaceID = firstNonEmpty(scope.WorkspaceID, strings.TrimSpace(project.WorkspaceID))
		return scope
	}
	return scope
}

func buildDeniedAuthorizationDecision(req unifiedpolicy.Request, session Session, authErr *apiError, auditDetails map[string]any) unifiedpolicy.Decision {
	return unifiedpolicy.Decision{
		Allowed:    false,
		Result:     "denied",
		Reason:     authErr.code,
		StatusCode: authErr.status,
		Code:       authErr.code,
		Message:    authErr.message,
		Details:    cloneMapAny(authErr.details),
		Subject:    subjectFromSession(session),
		AuditEvent: unifiedpolicy.AuditEvent{
			WorkspaceID: firstNonEmpty(req.WorkspaceID, session.WorkspaceID, localWorkspaceID),
			ActorID:     firstNonEmpty(session.UserID, "anonymous"),
			Action:      "authz." + req.Action,
			TargetType:  "resource",
			TargetID:    authorizationAuditTargetID(req.Resource),
			Result:      "denied",
			TraceID:     req.TraceID,
			Details:     auditDetails,
		},
	}
}

func subjectFromSession(session Session) unifiedpolicy.Subject {
	if strings.TrimSpace(session.UserID) == "" && strings.TrimSpace(session.WorkspaceID) == "" {
		return unifiedpolicy.Subject{}
	}
	roles := []string{}
	if strings.TrimSpace(string(session.Role)) != "" {
		roles = append(roles, string(session.Role))
	}
	return unifiedpolicy.Subject{
		ID:          strings.TrimSpace(session.UserID),
		WorkspaceID: strings.TrimSpace(session.WorkspaceID),
		DisplayName: strings.TrimSpace(session.DisplayName),
		Roles:       roles,
		Token:       strings.TrimSpace(session.Token),
	}
}

func sessionFromSubject(subject unifiedpolicy.Subject) Session {
	role := RoleDeveloper
	if len(subject.Roles) > 0 {
		role = parseRole(subject.Roles[0])
	}
	return Session{
		Token:       strings.TrimSpace(subject.Token),
		WorkspaceID: strings.TrimSpace(subject.WorkspaceID),
		Role:        role,
		UserID:      strings.TrimSpace(subject.ID),
		DisplayName: strings.TrimSpace(subject.DisplayName),
	}
}

func authorizationAuditTargetID(resource unifiedpolicy.Resource) string {
	return firstNonEmpty(
		strings.TrimSpace(resource.TargetID),
		strings.TrimSpace(resource.OwnerUserID),
		strings.TrimSpace(resource.WorkspaceID),
		"unknown",
	)
}

func recordBusinessOperationAudit(ctx context.Context, state *AppState, session Session, action string, targetType string, targetID string, details map[string]any) {
	if state == nil {
		return
	}
	event := unifiedpolicy.AuditEvent{
		WorkspaceID: firstNonEmpty(strings.TrimSpace(session.WorkspaceID), localWorkspaceID),
		ActorID:     firstNonEmpty(strings.TrimSpace(session.UserID), "anonymous"),
		Action:      strings.TrimSpace(action),
		TargetType:  firstNonEmpty(strings.TrimSpace(targetType), "resource"),
		TargetID:    firstNonEmpty(strings.TrimSpace(targetID), "unknown"),
		Result:      "success",
		TraceID:     TraceIDFromContext(ctx),
		Details:     cloneMapAny(details),
	}
	if state.unifiedPermissionGate != nil {
		_ = state.unifiedPermissionGate.RecordOperation(ctx, event)
		return
	}
	_ = recordAuthorizationAudit(state, event)
}
