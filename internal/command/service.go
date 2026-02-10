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
	"time"
)

type AuthzHook interface {
	Check(ctx context.Context, reqCtx RequestContext, cmd Command, permission string) (allowed bool, reason string, err error)
}

type Service struct {
	repo                 Repository
	idempotencyTTL       time.Duration
	allowPrivateToPublic bool
	logger               *log.Logger
	rbacHook             AuthzHook
	egressHook           AuthzHook
}

func NewService(repo Repository, idempotencyTTL time.Duration, allowPrivateToPublic bool, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		repo:                 repo,
		idempotencyTTL:       idempotencyTTL,
		allowPrivateToPublic: allowPrivateToPublic,
		logger:               logger,
	}
}

func (s *Service) SetRBACHook(hook AuthzHook) {
	s.rbacHook = hook
}

func (s *Service) SetEgressHook(hook AuthzHook) {
	s.egressHook = hook
}

func (s *Service) Submit(ctx context.Context, reqCtx RequestContext, commandType string, payload json.RawMessage, idempotencyKey, requestedVisibility string) (Command, error) {
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
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, created.Command.ID, "command.authorize", "allow", "stub_authorizer", payload)

	running, err := s.repo.SetStatus(ctx, reqCtx, created.Command.ID, StatusRunning, nil, "", "", nil)
	if err != nil {
		return Command{}, err
	}
	_ = s.repo.AppendCommandEvent(ctx, reqCtx, running.ID, "command.running", payload)

	result := map[string]any{
		"handled":     true,
		"commandType": commandType,
		"executedAt":  time.Now().UTC().Format(time.RFC3339Nano),
	}
	resultBytes, _ := json.Marshal(result)
	finishedAt := time.Now().UTC()
	final, err := s.repo.SetStatus(ctx, reqCtx, running.ID, StatusSucceeded, resultBytes, "", "", &finishedAt)
	if err != nil {
		return Command{}, err
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, final.ID, "command.succeeded", resultBytes)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, final.ID, "command.execute", "allow", "stub_execute", resultBytes)

	return final, nil
}

func (s *Service) Get(ctx context.Context, reqCtx RequestContext, id string) (Command, error) {
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
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	subjectType = strings.ToLower(strings.TrimSpace(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	resourceID = strings.TrimSpace(resourceID)

	if resourceType != "command" || resourceID == "" || subjectType != "user" || subjectID == "" {
		return Share{}, ErrInvalidShareRequest
	}

	normalizedPermissions, err := normalizePermissions(permissions)
	if err != nil {
		return Share{}, err
	}

	cmd, err := s.repo.GetForAccess(ctx, reqCtx, resourceID)
	if err != nil {
		if errors.Is(err, ErrNotImplemented) {
			return Share{}, err
		}
		return Share{}, ErrInvalidShareRequest
	}

	allowed, reason, err := s.authorizeCommand(ctx, reqCtx, cmd, PermissionShare)
	if err != nil {
		return Share{}, err
	}
	if !allowed {
		_ = s.repo.AppendAuditEvent(ctx, reqCtx, resourceID, "share.create", "deny", reason, nil)
		return Share{}, &ForbiddenError{Reason: reason}
	}

	created, err := s.repo.CreateShare(ctx, ShareCreateInput{
		Context:      reqCtx,
		ResourceType: resourceType,
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
	return s.repo.ListShares(ctx, params)
}

func (s *Service) DeleteShare(ctx context.Context, reqCtx RequestContext, shareID string) error {
	return s.repo.DeleteShare(ctx, reqCtx, shareID)
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
