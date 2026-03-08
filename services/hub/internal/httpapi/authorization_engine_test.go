package httpapi

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	unifiedpolicy "goyais/services/hub/internal/agent/policy/unified"
)

type unifiedAuthorizationGateStub struct {
	called      bool
	lastRequest unifiedpolicy.Request
	decision    unifiedpolicy.Decision
	err         error
}

func (s *unifiedAuthorizationGateStub) Authorize(_ context.Context, req unifiedpolicy.Request) (unifiedpolicy.Decision, error) {
	s.called = true
	s.lastRequest = req
	if s.err != nil {
		return unifiedpolicy.Decision{}, s.err
	}
	return s.decision, nil
}

func (s *unifiedAuthorizationGateStub) RecordOperation(_ context.Context, _ unifiedpolicy.AuditEvent) error {
	return nil
}

func TestAuthorizeActionUsesUnifiedPermissionGateWhenConfigured(t *testing.T) {
	state := NewAppState(nil)
	stub := &unifiedAuthorizationGateStub{
		decision: unifiedpolicy.Decision{
			Allowed: true,
			Result:  "success",
			Subject: unifiedpolicy.Subject{
				ID:          "local_user",
				WorkspaceID: localWorkspaceID,
				DisplayName: "Local User",
				Roles:       []string{string(RoleAdmin)},
			},
		},
	}
	state.unifiedPermissionGate = stub

	req := httptest.NewRequest("GET", "/v1/sessions", nil)
	session, err := authorizeAction(
		state,
		req,
		localWorkspaceID,
		"session.read",
		authorizationResource{WorkspaceID: localWorkspaceID},
		authorizationContext{OperationType: "read"},
	)
	if err != nil {
		t.Fatalf("expected no authz error, got %#v", err)
	}
	if !stub.called {
		t.Fatalf("expected unified permission gate to be called")
	}
	if stub.lastRequest.Action != "session.read" {
		t.Fatalf("expected action session.read, got %#v", stub.lastRequest)
	}
	if session.WorkspaceID != localWorkspaceID {
		t.Fatalf("expected local session workspace, got %#v", session)
	}
}

func TestAuthorizeActionRecordsPermissionRequestHookAudit(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_authz_hook_audit"
	sessionID := "conv_authz_hook_audit"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Authz Hook Audit Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Authz Hook Audit Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_authz_hook_audit",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.adminAudit = nil

	req := httptest.NewRequest("GET", "/v1/sessions/"+sessionID, nil)
	if _, err := authorizeAction(
		state,
		req,
		localWorkspaceID,
		"session.read",
		authorizationResource{
			WorkspaceID: localWorkspaceID,
			TargetID:    sessionID,
		},
		authorizationContext{OperationType: "read"},
	); err != nil {
		t.Fatalf("expected no authz error, got %#v", err)
	}

	for _, entry := range state.adminAudit {
		if entry.Action != "hook.permission_request" {
			continue
		}
		if entry.Resource != sessionID {
			t.Fatalf("expected hook.permission_request resource %q, got %#v", sessionID, entry)
		}
		if entry.Result != string(HookDecisionActionAllow) {
			t.Fatalf("expected hook.permission_request result allow, got %#v", entry)
		}
		return
	}
	t.Fatalf("expected hook.permission_request audit entry, got %#v", state.adminAudit)
}

func TestAuthorizeActionPermissionRequestHookAuditUsesMatchingHookPolicy(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_authz_hook_policy"
	sessionID := "conv_authz_hook_policy"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Authz Hook Policy Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Authz Hook Policy Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_authz_hook_policy",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.hookPolicies["policy_permission_request_deny"] = HookPolicy{
		ID:          "policy_permission_request_deny",
		Scope:       HookScopeLocal,
		Event:       HookEventTypePermissionRequest,
		HandlerType: HookHandlerTypeAgent,
		ToolName:    "session.read",
		SessionID:   sessionID,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "approval required",
		},
		UpdatedAt: now,
	}
	state.adminAudit = nil

	req := httptest.NewRequest("GET", "/v1/sessions/"+sessionID, nil)
	if _, err := authorizeAction(
		state,
		req,
		localWorkspaceID,
		"session.read",
		authorizationResource{
			WorkspaceID: localWorkspaceID,
			TargetID:    sessionID,
		},
		authorizationContext{OperationType: "read"},
	); err != nil {
		t.Fatalf("expected authz decision to stay allowed, got %#v", err)
	}

	for _, entry := range state.adminAudit {
		if entry.Action != "hook.permission_request" {
			continue
		}
		if entry.Resource != sessionID {
			t.Fatalf("expected hook.permission_request resource %q, got %#v", sessionID, entry)
		}
		if entry.Result != string(HookDecisionActionDeny) {
			t.Fatalf("expected hook.permission_request result deny, got %#v", entry)
		}
		return
	}
	t.Fatalf("expected hook.permission_request audit entry, got %#v", state.adminAudit)
}
