package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"goyais/internal/platform/eventbus"
)

type commandExecutionContextKey string

const commandExecutionKeyID commandExecutionContextKey = "command_id"

func CurrentCommandID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(commandExecutionKeyID).(string)
	return strings.TrimSpace(value)
}

type AuthzHook interface {
	Check(ctx context.Context, reqCtx RequestContext, cmd Command, permission string) (allowed bool, reason string, err error)
}

type ExecuteFunc func(ctx context.Context, reqCtx RequestContext, payload json.RawMessage) (result []byte, err error)

type Service struct {
	repo                  Repository
	idempotencyTTL        time.Duration
	allowPrivateToPublic  bool
	aclRoleSubjectEnabled bool
	logger                *log.Logger
	rbacHook              AuthzHook
	egressHook            AuthzHook
	eventBus              eventbus.Provider
	executorsMu           sync.RWMutex
	executors             map[string]ExecuteFunc
}

func NewService(repo Repository, idempotencyTTL time.Duration, allowPrivateToPublic bool, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		repo:                  repo,
		idempotencyTTL:        idempotencyTTL,
		allowPrivateToPublic:  allowPrivateToPublic,
		aclRoleSubjectEnabled: true,
		logger:                logger,
		executors:             make(map[string]ExecuteFunc),
	}
}

func (s *Service) SetRBACHook(hook AuthzHook) {
	s.rbacHook = hook
}

func (s *Service) SetEgressHook(hook AuthzHook) {
	s.egressHook = hook
}

func (s *Service) SetACLRoleSubjectEnabled(enabled bool) {
	s.aclRoleSubjectEnabled = enabled
}

func (s *Service) SetEventBusProvider(provider eventbus.Provider) {
	s.eventBus = provider
}

func (s *Service) SetExecutor(commandType string, executor ExecuteFunc) {
	key := strings.TrimSpace(commandType)
	if key == "" || executor == nil {
		return
	}

	s.executorsMu.Lock()
	defer s.executorsMu.Unlock()
	s.executors[key] = executor
}

func (s *Service) Submit(ctx context.Context, reqCtx RequestContext, commandType string, payload json.RawMessage, idempotencyKey, requestedVisibility string) (Command, error) {
	reqCtx = s.normalizeRequestContext(reqCtx)
	if err := validateCreateRequest(commandType, payload); err != nil {
		return Command{}, err
	}

	visibility, err := s.normalizeCommandVisibility(requestedVisibility)
	if err != nil {
		return Command{}, err
	}

	now := time.Now().UTC()
	requestHash := ""
	if idempotencyKey != "" {
		requestHash = hashRequest(commandType, payload)
	} else {
		s.logger.Printf("WARN: missing Idempotency-Key tenant=%s workspace=%s user=%s", reqCtx.TenantID, reqCtx.WorkspaceID, reqCtx.UserID)
	}

	created, err := s.repo.Create(ctx, CreateInput{
		Context:        reqCtx,
		CommandType:    commandType,
		Payload:        payload,
		Visibility:     visibility,
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
		Now:            now,
		TTL:            s.idempotencyTTL,
	})
	if err != nil {
		return Command{}, err
	}

	if created.Reused {
		return created.Command, nil
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, created.Command.ID, "command.accepted", payload)
	s.publishCommandEvent(ctx, reqCtx, created.Command.ID, "command.accepted", payload)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, created.Command.ID, "command.authorize", "allow", "stub_authorizer", payload)

	running, err := s.repo.SetStatus(ctx, reqCtx, created.Command.ID, StatusRunning, nil, "", "", nil)
	if err != nil {
		return Command{}, err
	}
	_ = s.repo.AppendCommandEvent(ctx, reqCtx, running.ID, "command.running", payload)

	executionCtx := context.WithValue(ctx, commandExecutionKeyID, running.ID)
	resultBytes, err := s.executeCommand(executionCtx, reqCtx, commandType, payload)
	if err != nil {
		failed, markErr := s.markCommandFailed(ctx, reqCtx, running.ID, commandType, payload, resultBytes, err)
		if markErr != nil {
			return Command{}, markErr
		}
		return failed, err
	}

	finishedAt := time.Now().UTC()
	final, err := s.repo.SetStatus(ctx, reqCtx, running.ID, StatusSucceeded, resultBytes, "", "", &finishedAt)
	if err != nil {
		return Command{}, err
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, final.ID, "command.succeeded", resultBytes)
	s.publishCommandEvent(ctx, reqCtx, final.ID, "command.succeeded", resultBytes)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, final.ID, "command.execute", "allow", "stub_execute", resultBytes)
	_ = s.repo.AppendAuditEvent(
		ctx,
		reqCtx,
		final.ID,
		"command.egress",
		"allow",
		"policy.default_allow",
		buildEgressAuditPayload(commandType, payload, resultBytes, "allow"),
	)
	if streamEventType := resolveStreamAuditEventType(commandType); streamEventType != "" {
		_ = s.repo.AppendAuditEvent(ctx, reqCtx, final.ID, streamEventType, "allow", "command_succeeded", resultBytes)
	}

	return final, nil
}

func resolveStreamAuditEventType(commandType string) string {
	switch strings.TrimSpace(commandType) {
	case "stream.updateAuth":
		return "stream.auth.updated"
	case "stream.delete":
		return "stream.deleted"
	default:
		return ""
	}
}

func (s *Service) executeCommand(ctx context.Context, reqCtx RequestContext, commandType string, payload json.RawMessage) ([]byte, error) {
	s.executorsMu.RLock()
	executor := s.executors[commandType]
	s.executorsMu.RUnlock()

	if executor == nil {
		return defaultCommandResult(commandType), nil
	}

	result, err := executor(ctx, reqCtx, payload)
	if err != nil {
		return result, err
	}
	if len(result) == 0 {
		return defaultCommandResult(commandType), nil
	}
	return result, nil
}

func (s *Service) markCommandFailed(
	ctx context.Context,
	reqCtx RequestContext,
	commandID string,
	commandType string,
	commandPayload json.RawMessage,
	result []byte,
	execErr error,
) (Command, error) {
	errorCode := "COMMAND_EXECUTION_FAILED"
	messageKey := "error.command.execution_failed"
	reason := "executor_failed"
	eventPayload := result

	var execMeta *ExecutionError
	if errors.As(execErr, &execMeta) {
		if strings.TrimSpace(execMeta.Code) != "" {
			errorCode = strings.TrimSpace(execMeta.Code)
		}
		if strings.TrimSpace(execMeta.MessageKey) != "" {
			messageKey = strings.TrimSpace(execMeta.MessageKey)
		}
	}
	if len(eventPayload) == 0 {
		fallback, _ := json.Marshal(map[string]any{
			"errorCode":  errorCode,
			"messageKey": messageKey,
		})
		eventPayload = fallback
	}
	if strings.TrimSpace(execErr.Error()) != "" {
		reason = execErr.Error()
	}

	finishedAt := time.Now().UTC()
	failed, err := s.repo.SetStatus(ctx, reqCtx, commandID, StatusFailed, eventPayload, errorCode, messageKey, &finishedAt)
	if err != nil {
		return Command{}, err
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, failed.ID, "command.failed", eventPayload)
	s.publishCommandEvent(ctx, reqCtx, failed.ID, "command.failed", eventPayload)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, failed.ID, "command.execute", "deny", reason, eventPayload)
	_ = s.repo.AppendAuditEvent(
		ctx,
		reqCtx,
		failed.ID,
		"command.egress",
		"deny",
		reason,
		buildEgressAuditPayload(commandType, commandPayload, eventPayload, "deny"),
	)
	return failed, nil
}

func (s *Service) publishCommandEvent(ctx context.Context, reqCtx RequestContext, commandID, eventType string, payload []byte) {
	if s.eventBus == nil {
		return
	}
	envelope := map[string]any{
		"eventType":   eventType,
		"commandId":   commandID,
		"tenantId":    reqCtx.TenantID,
		"workspaceId": reqCtx.WorkspaceID,
		"userId":      reqCtx.UserID,
		"traceId":     reqCtx.TraceID,
		"emittedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	}
	if len(payload) > 0 {
		var parsed any
		if err := json.Unmarshal(payload, &parsed); err == nil {
			envelope["payload"] = parsed
		} else {
			envelope["payloadRaw"] = string(payload)
		}
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	err = s.eventBus.Publish(ctx, eventbus.ChannelCommand, eventbus.Message{
		Key:   commandID,
		Value: raw,
		Headers: map[string]string{
			"eventType":   eventType,
			"tenantId":    reqCtx.TenantID,
			"workspaceId": reqCtx.WorkspaceID,
		},
	})
	auditPayload, _ := json.Marshal(map[string]any{
		"channel":   eventbus.ChannelCommand,
		"eventType": eventType,
	})
	if err != nil {
		_ = s.repo.AppendAuditEvent(ctx, reqCtx, commandID, "command.eventbus", "deny", err.Error(), auditPayload)
		return
	}
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, commandID, "command.eventbus", "allow", "published", auditPayload)
}

func buildEgressAuditPayload(commandType string, requestPayload json.RawMessage, responsePayload []byte, policyResult string) []byte {
	policyResult = strings.ToLower(strings.TrimSpace(policyResult))
	if policyResult == "" {
		policyResult = "unknown"
	}
	summary := map[string]any{
		"commandType":   strings.TrimSpace(commandType),
		"requestBytes":  len(requestPayload),
		"requestDigest": hashRequest(commandType, requestPayload),
	}
	if len(responsePayload) > 0 {
		hash := sha256.Sum256(responsePayload)
		summary["responseBytes"] = len(responsePayload)
		summary["responseDigest"] = hex.EncodeToString(hash[:])
	}

	payload := map[string]any{
		"destination":  "local://command-executor",
		"policyResult": policyResult,
		"summary":      summary,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"destination":"local://command-executor","policyResult":"unknown","summary":{}}`)
	}
	return raw
}

func defaultCommandResult(commandType string) []byte {
	result := map[string]any{
		"handled":     true,
		"commandType": commandType,
		"executedAt":  time.Now().UTC().Format(time.RFC3339Nano),
	}
	payload, _ := json.Marshal(result)
	return payload
}

func (s *Service) Get(ctx context.Context, reqCtx RequestContext, id string) (Command, error) {
	reqCtx = s.normalizeRequestContext(reqCtx)
	cmd, err := s.repo.GetForAccess(ctx, reqCtx, id)
	if err != nil {
		return Command{}, err
	}

	allowed, reason, err := s.authorizeCommand(ctx, reqCtx, cmd, PermissionRead)
	if err != nil {
		return Command{}, err
	}
	if !allowed {
		_ = s.repo.AppendAuditEvent(ctx, reqCtx, cmd.ID, "command.read", "deny", reason, nil)
		return Command{}, &ForbiddenError{Reason: reason}
	}

	_ = s.repo.AppendAuditEvent(ctx, reqCtx, cmd.ID, "command.read", "allow", "authorized", nil)
	return cmd, nil
}

func (s *Service) List(ctx context.Context, params ListParams) (ListResult, error) {
	params.Context = s.normalizeRequestContext(params.Context)
	return s.repo.List(ctx, params)
}

func (s *Service) CreateShare(
	ctx context.Context,
	reqCtx RequestContext,
	resourceType string,
	resourceID string,
	subjectType string,
	subjectID string,
	permissions []string,
	expiresAt *time.Time,
) (Share, error) {
	reqCtx = s.normalizeRequestContext(reqCtx)
	normalizedResourceType, err := normalizeShareResourceType(resourceType)
	if err != nil {
		return Share{}, err
	}
	subjectType = strings.ToLower(strings.TrimSpace(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	resourceID = strings.TrimSpace(resourceID)

	if resourceID == "" || subjectID == "" {
		return Share{}, ErrInvalidShareRequest
	}
	if subjectType != "user" && subjectType != "role" {
		return Share{}, ErrInvalidShareRequest
	}
	if subjectType == "role" && !s.aclRoleSubjectEnabled {
		return Share{}, ErrInvalidShareRequest
	}

	normalizedPermissions, err := normalizePermissions(permissions)
	if err != nil {
		return Share{}, err
	}

	resource, err := s.repo.GetShareResource(ctx, reqCtx, normalizedResourceType, resourceID)
	if err != nil {
		if errors.Is(err, ErrNotImplemented) {
			return Share{}, err
		}
		return Share{}, ErrInvalidShareRequest
	}

	allowed := reqCtx.UserID == resource.OwnerID
	reason := "authorized"
	if !allowed {
		hasPermission, err := s.repo.HasShareResourcePermission(ctx, reqCtx, normalizedResourceType, resourceID, PermissionShare, time.Now().UTC())
		if err != nil {
			return Share{}, err
		}
		if hasPermission {
			allowed = true
		}
	}
	if !allowed {
		reason = "permission_denied"
		_ = s.repo.AppendAuditEvent(ctx, reqCtx, resourceID, "share.create", "deny", reason, nil)
		return Share{}, &ForbiddenError{Reason: reason}
	}

	created, err := s.repo.CreateShare(ctx, ShareCreateInput{
		Context:      reqCtx,
		ResourceType: normalizedResourceType,
		ResourceID:   resourceID,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		Permissions:  normalizedPermissions,
		ExpiresAt:    expiresAt,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return Share{}, err
	}

	_ = s.repo.AppendAuditEvent(ctx, reqCtx, resourceID, "share.create", "allow", "owner_or_share_permission", nil)
	return created, nil
}

func (s *Service) ListShares(ctx context.Context, params ShareListParams) (ShareListResult, error) {
	params.Context = s.normalizeRequestContext(params.Context)
	return s.repo.ListShares(ctx, params)
}

func (s *Service) DeleteShare(ctx context.Context, reqCtx RequestContext, shareID string) error {
	reqCtx = s.normalizeRequestContext(reqCtx)
	return s.repo.DeleteShare(ctx, reqCtx, shareID)
}

func (s *Service) normalizeRequestContext(reqCtx RequestContext) RequestContext {
	if s.aclRoleSubjectEnabled {
		return reqCtx
	}
	reqCtx.Roles = nil
	return reqCtx
}

func (s *Service) authorizeCommand(ctx context.Context, reqCtx RequestContext, cmd Command, permission string) (bool, string, error) {
	if strings.TrimSpace(reqCtx.TenantID) == "" || reqCtx.TenantID != cmd.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(reqCtx.WorkspaceID) == "" || reqCtx.WorkspaceID != cmd.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if reqCtx.UserID == cmd.OwnerID {
		allowed = true
	}
	if !allowed && permission == PermissionRead && cmd.Visibility == VisibilityWorkspace {
		allowed = true
	}

	if !allowed {
		hasACL, err := s.repo.HasCommandPermission(ctx, reqCtx, cmd.ID, permission, time.Now().UTC())
		if err != nil {
			return false, "", err
		}
		allowed = hasACL
	}
	if !allowed {
		return false, "permission_denied", nil
	}

	if s.rbacHook != nil {
		ok, reason, err := s.rbacHook.Check(ctx, reqCtx, cmd, permission)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return false, reasonOrDefault(reason, "rbac_denied"), nil
		}
	}

	if s.egressHook != nil {
		ok, reason, err := s.egressHook.Check(ctx, reqCtx, cmd, permission)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return false, reasonOrDefault(reason, "egress_denied"), nil
		}
	}

	return true, "authorized", nil
}

func (s *Service) normalizeCommandVisibility(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return VisibilityPrivate, nil
	}

	switch value {
	case VisibilityPrivate:
		return value, nil
	case VisibilityWorkspace:
		if !s.allowPrivateToPublic {
			return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
		}
		return value, nil
	case VisibilityTenant, VisibilityPublic:
		return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
	default:
		return "", ErrInvalidCommandRequest
	}
}

func normalizePermissions(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidShareRequest
	}

	allowed := map[string]struct{}{
		PermissionRead:    {},
		PermissionWrite:   {},
		PermissionExecute: {},
		PermissionManage:  {},
		PermissionShare:   {},
	}

	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		value := strings.ToUpper(strings.TrimSpace(item))
		if value == "" {
			return nil, ErrInvalidShareRequest
		}
		if _, ok := allowed[value]; !ok {
			return nil, ErrInvalidShareRequest
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	if len(result) == 0 {
		return nil, ErrInvalidShareRequest
	}
	sort.Strings(result)
	return result, nil
}

func normalizeShareResourceType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "command", "asset":
		return value, nil
	default:
		return "", ErrInvalidShareRequest
	}
}

func reasonOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func validateCreateRequest(commandType string, payload json.RawMessage) error {
	if strings.TrimSpace(commandType) == "" {
		return ErrInvalidCommandRequest
	}
	if len(payload) == 0 {
		return ErrInvalidCommandRequest
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return ErrInvalidCommandRequest
	}

	return nil
}

func hashRequest(commandType string, payload json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(commandType))
	h.Write([]byte("\n"))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
