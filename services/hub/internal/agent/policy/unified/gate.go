package unified

import (
	"context"
	"strings"

	"goyais/services/hub/internal/agent/core"
)

type Subject struct {
	ID          string
	WorkspaceID string
	DisplayName string
	Roles       []string
	Token       string
}

type Resource struct {
	WorkspaceID  string
	OwnerUserID  string
	Scope        string
	ResourceType string
	ShareStatus  string
	TargetID     string
}

type Context struct {
	RiskLevel     string
	OperationType string
	RequestSource string
	ABACRequired  bool
}

type Request struct {
	Action            string
	WorkspaceID       string
	AccessToken       string
	TraceID           string
	AllowedRoles      []string
	Subject           Subject
	Resource          Resource
	Context           Context
	PermissionRequest *core.PermissionRequest
}

type AuditEvent struct {
	WorkspaceID string
	ActorID     string
	Action      string
	TargetType  string
	TargetID    string
	Result      string
	TraceID     string
	Details     map[string]any
}

type Decision struct {
	Allowed    bool
	Result     string
	Reason     string
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
	Subject    Subject
	AuditEvent AuditEvent
}

type Evaluator interface {
	Evaluate(ctx context.Context, req Request) (Decision, error)
}

type AuditLogger interface {
	Record(ctx context.Context, event AuditEvent) error
}

type HookObserver interface {
	Observe(ctx context.Context, req Request, decision Decision) error
}

type Options struct {
	AuditLogger    AuditLogger
	HookObserver   HookObserver
	PermissionGate core.PermissionGate
}

type Gate struct {
	evaluator      Evaluator
	auditLogger    AuditLogger
	hookObserver   HookObserver
	permissionGate core.PermissionGate
}

func NewGate(evaluator Evaluator, options Options) *Gate {
	return &Gate{
		evaluator:      evaluator,
		auditLogger:    options.AuditLogger,
		hookObserver:   options.HookObserver,
		permissionGate: options.PermissionGate,
	}
}

func (g *Gate) Authorize(ctx context.Context, req Request) (Decision, error) {
	if g == nil || g.evaluator == nil {
		return Decision{
			Allowed: true,
			Result:  "success",
			Subject: req.Subject,
		}, nil
	}

	decision, err := g.evaluator.Evaluate(ctx, req)
	if err != nil {
		return Decision{}, err
	}
	decision = normalizeDecision(req, decision)

	if decision.Allowed && g.permissionGate != nil && req.PermissionRequest != nil {
		permissionDecision, permissionErr := g.permissionGate.Evaluate(ctx, *req.PermissionRequest)
		if permissionErr != nil {
			return Decision{}, permissionErr
		}
		if permissionDecision.Kind != core.PermissionDecisionAllow {
			decision.Allowed = false
			decision.Result = "denied"
			decision.Reason = strings.TrimSpace(permissionDecision.Reason)
			if decision.Reason == "" {
				decision.Reason = "permission gate denied request"
			}
			if decision.StatusCode == 0 {
				decision.StatusCode = 403
			}
			if strings.TrimSpace(decision.Code) == "" {
				decision.Code = "ACCESS_DENIED"
			}
			if strings.TrimSpace(decision.Message) == "" {
				decision.Message = "Permission is denied"
			}
			decision.Details = cloneMapWith(decision.Details, map[string]any{
				"permission_decision": string(permissionDecision.Kind),
				"matched_rule":        strings.TrimSpace(permissionDecision.MatchedRule),
			})
			decision.AuditEvent.Result = "denied"
			decision.AuditEvent.Details = cloneMapWith(decision.AuditEvent.Details, map[string]any{
				"permission_decision": string(permissionDecision.Kind),
				"matched_rule":        strings.TrimSpace(permissionDecision.MatchedRule),
				"reason":              decision.Reason,
			})
		}
	}

	if g.auditLogger != nil {
		_ = g.auditLogger.Record(ctx, decision.AuditEvent)
	}
	if g.hookObserver != nil {
		_ = g.hookObserver.Observe(ctx, req, decision)
	}
	return decision, nil
}

func (g *Gate) RecordOperation(ctx context.Context, event AuditEvent) error {
	if g == nil || g.auditLogger == nil {
		return nil
	}
	normalized := event
	if strings.TrimSpace(normalized.Result) == "" {
		normalized.Result = "success"
	}
	if normalized.Details == nil {
		normalized.Details = map[string]any{}
	}
	return g.auditLogger.Record(ctx, normalized)
}

func normalizeDecision(req Request, decision Decision) Decision {
	if strings.TrimSpace(decision.Result) == "" {
		if decision.Allowed {
			decision.Result = "success"
		} else {
			decision.Result = "denied"
		}
	}
	if strings.TrimSpace(decision.AuditEvent.WorkspaceID) == "" {
		decision.AuditEvent.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	}
	if strings.TrimSpace(decision.AuditEvent.Action) == "" {
		decision.AuditEvent.Action = strings.TrimSpace(req.Action)
	}
	if strings.TrimSpace(decision.AuditEvent.Result) == "" {
		decision.AuditEvent.Result = strings.TrimSpace(decision.Result)
	}
	if strings.TrimSpace(decision.AuditEvent.TraceID) == "" {
		decision.AuditEvent.TraceID = strings.TrimSpace(req.TraceID)
	}
	if decision.AuditEvent.Details == nil {
		decision.AuditEvent.Details = map[string]any{}
	}
	if decision.Subject.ID == "" {
		decision.Subject = req.Subject
	}
	return decision
}

func cloneMapWith(base map[string]any, extra map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range base {
		result[key] = value
	}
	for key, value := range extra {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = value
	}
	return result
}
