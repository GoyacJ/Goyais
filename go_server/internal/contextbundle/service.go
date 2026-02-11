// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package contextbundle

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"goyais/internal/command"
)

type Service struct {
	repo                 Repository
	allowPrivateToPublic bool
}

func NewService(repo Repository, allowPrivateToPublic bool) *Service {
	return &Service{
		repo:                 repo,
		allowPrivateToPublic: allowPrivateToPublic,
	}
}

func (s *Service) ListBundles(ctx context.Context, params ListParams) (ListResult, error) {
	return s.repo.ListBundles(ctx, params)
}

func (s *Service) GetBundle(ctx context.Context, req command.RequestContext, bundleID string) (Bundle, error) {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return Bundle{}, ErrInvalidRequest
	}
	item, err := s.repo.GetBundleForAccess(ctx, req, bundleID)
	if err != nil {
		return Bundle{}, err
	}
	allowed, reason, err := s.authorizeBundle(ctx, req, item, command.PermissionRead)
	if err != nil {
		return Bundle{}, err
	}
	if !allowed {
		return Bundle{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) RebuildBundle(
	ctx context.Context,
	req command.RequestContext,
	scopeType,
	scopeID,
	visibility string,
) (Bundle, error) {
	normalizedScope, normalizedScopeID, err := normalizeScope(scopeType, scopeID, req.WorkspaceID)
	if err != nil {
		return Bundle{}, err
	}
	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return Bundle{}, err
	}

	now := time.Now().UTC()
	facts, _ := json.Marshal(map[string]any{
		"scopeType":  normalizedScope,
		"scopeId":    normalizedScopeID,
		"requestedBy": req.UserID,
		"rebuiltAt":  now.Format(time.RFC3339Nano),
	})
	summaries, _ := json.Marshal(map[string]any{
		"text": "context bundle rebuilt",
	})
	refs, _ := json.Marshal(map[string]any{
		"commands": []any{},
		"assets":   []any{},
	})
	embeddings, _ := json.Marshal([]any{})
	timeline, _ := json.Marshal([]map[string]any{ {
		"ts":   now.Format(time.RFC3339Nano),
		"type": "context.bundle.rebuild",
		"desc": "bundle rebuilt",
	}})

	return s.repo.UpsertBundle(ctx, RebuildInput{
		Context:             req,
		ScopeType:           normalizedScope,
		ScopeID:             normalizedScopeID,
		Visibility:          normalizedVisibility,
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: embeddings,
		Timeline:            timeline,
		Now:                 now,
	})
}

func (s *Service) authorizeBundle(
	ctx context.Context,
	req command.RequestContext,
	item Bundle,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != item.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != item.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == item.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && item.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasBundlePermission(ctx, req, item.ID, permission, time.Now().UTC())
		if err != nil {
			return false, "", err
		}
		allowed = hasPermission
	}
	if !allowed {
		return false, "permission_denied", nil
	}
	return true, "authorized", nil
}

func (s *Service) normalizeVisibility(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return command.VisibilityPrivate, nil
	}
	switch value {
	case command.VisibilityPrivate, command.VisibilityWorkspace:
		return value, nil
	case command.VisibilityTenant, command.VisibilityPublic:
		if s.allowPrivateToPublic {
			return value, nil
		}
		return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeScope(rawScope, rawScopeID, workspaceID string) (string, string, error) {
	scopeType := strings.ToLower(strings.TrimSpace(rawScope))
	scopeID := strings.TrimSpace(rawScopeID)
	switch scopeType {
	case ScopeTypeRun, ScopeTypeSession:
		if scopeID == "" {
			return "", "", ErrInvalidRequest
		}
		return scopeType, scopeID, nil
	case ScopeTypeWorkspace, "":
		if scopeID == "" {
			scopeID = strings.TrimSpace(workspaceID)
		}
		if scopeID == "" {
			return "", "", ErrInvalidRequest
		}
		return ScopeTypeWorkspace, scopeID, nil
	default:
		return "", "", ErrInvalidRequest
	}
}
