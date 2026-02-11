// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

const (
	RunModeSync    = "sync"
	RunModeRunning = "running"
	RunModeFail    = "fail"
	RunModeRetry   = "retry"
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

func (s *Service) CreateTemplateDraft(
	ctx context.Context,
	req command.RequestContext,
	name string,
	description string,
	graph json.RawMessage,
	schemaInputs json.RawMessage,
	schemaOutputs json.RawMessage,
	uiState json.RawMessage,
	visibility string,
) (WorkflowTemplate, error) {
	if strings.TrimSpace(name) == "" || !isJSONObject(graph) {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return WorkflowTemplate{}, err
	}

	if len(schemaInputs) == 0 {
		schemaInputs = json.RawMessage(`{}`)
	}
	if len(schemaOutputs) == 0 {
		schemaOutputs = json.RawMessage(`{}`)
	}
	if len(uiState) == 0 {
		uiState = json.RawMessage(`{}`)
	}

	return s.repo.CreateTemplate(ctx, CreateTemplateInput{
		Context:       req,
		Name:          strings.TrimSpace(name),
		Description:   strings.TrimSpace(description),
		Visibility:    normalizedVisibility,
		Graph:         graph,
		SchemaInputs:  schemaInputs,
		SchemaOutputs: schemaOutputs,
		UIState:       uiState,
		Now:           time.Now().UTC(),
	})
}

func (s *Service) PatchTemplate(
	ctx context.Context,
	req command.RequestContext,
	templateID string,
	patch json.RawMessage,
) (WorkflowTemplate, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" || !isJSONObject(patch) {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	tpl, err := s.repo.GetTemplateForAccess(ctx, req, templateID)
	if err != nil {
		return WorkflowTemplate{}, err
	}

	allowed, reason, err := s.authorizeTemplate(ctx, req, tpl, command.PermissionWrite)
	if err != nil {
		return WorkflowTemplate{}, err
	}
	if !allowed {
		return WorkflowTemplate{}, &ForbiddenError{Reason: reason}
	}

	if tpl.Status == TemplateStatusDisabled {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	var patchBody map[string]json.RawMessage
	if err := json.Unmarshal(patch, &patchBody); err != nil {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	nextGraph := tpl.GraphJSON
	if rawGraph, ok := patchBody["graph"]; ok {
		if !isJSONObject(rawGraph) {
			return WorkflowTemplate{}, ErrInvalidRequest
		}
		nextGraph = rawGraph
	} else {
		opsRaw, ok := patchBody["operations"]
		if !ok {
			return WorkflowTemplate{}, ErrInvalidRequest
		}
		var ops []json.RawMessage
		if err := json.Unmarshal(opsRaw, &ops); err != nil || len(ops) == 0 {
			return WorkflowTemplate{}, ErrInvalidRequest
		}
		patchedGraph, err := applyPatchOperationsToGraph(tpl.GraphJSON, opsRaw)
		if err != nil {
			return WorkflowTemplate{}, ErrInvalidRequest
		}
		nextGraph = patchedGraph
	}

	// Keep patch metadata in ui_state so clients can inspect the last patch payload.
	nextUIState, _ := json.Marshal(map[string]any{
		"lastPatch": json.RawMessage(patch),
	})

	return s.repo.PatchTemplate(ctx, PatchTemplateInput{
		Context:    req,
		TemplateID: templateID,
		Graph:      nextGraph,
		UIState:    nextUIState,
		Now:        time.Now().UTC(),
	})
}

func (s *Service) PublishTemplate(ctx context.Context, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	tpl, err := s.repo.GetTemplateForAccess(ctx, req, templateID)
	if err != nil {
		return WorkflowTemplate{}, err
	}
	allowed, reason, err := s.authorizeTemplate(ctx, req, tpl, command.PermissionManage)
	if err != nil {
		return WorkflowTemplate{}, err
	}
	if !allowed {
		return WorkflowTemplate{}, &ForbiddenError{Reason: reason}
	}

	if tpl.Status == TemplateStatusDisabled {
		return WorkflowTemplate{}, ErrInvalidRequest
	}

	return s.repo.PublishTemplate(ctx, PublishTemplateInput{
		Context:    req,
		TemplateID: templateID,
		Now:        time.Now().UTC(),
	})
}

func (s *Service) GetTemplate(ctx context.Context, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	tpl, err := s.repo.GetTemplateForAccess(ctx, req, strings.TrimSpace(templateID))
	if err != nil {
		return WorkflowTemplate{}, err
	}

	allowed, reason, err := s.authorizeTemplate(ctx, req, tpl, command.PermissionRead)
	if err != nil {
		return WorkflowTemplate{}, err
	}
	if !allowed {
		return WorkflowTemplate{}, &ForbiddenError{Reason: reason}
	}
	return tpl, nil
}

func (s *Service) ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResult, error) {
	return s.repo.ListTemplates(ctx, params)
}

func (s *Service) CreateRun(
	ctx context.Context,
	req command.RequestContext,
	templateID string,
	inputs json.RawMessage,
	visibility string,
	mode string,
	fromStepKey string,
	testNode bool,
) (WorkflowRun, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return WorkflowRun{}, ErrInvalidRequest
	}
	fromStepKey = strings.TrimSpace(fromStepKey)
	if len(inputs) == 0 {
		inputs = json.RawMessage(`{}`)
	}
	if !isJSONObject(inputs) {
		return WorkflowRun{}, ErrInvalidRequest
	}

	normalizedMode, err := normalizeRunMode(mode)
	if err != nil {
		return WorkflowRun{}, err
	}

	tpl, err := s.repo.GetTemplateForAccess(ctx, req, templateID)
	if err != nil {
		return WorkflowRun{}, err
	}
	allowed, reason, err := s.authorizeTemplate(ctx, req, tpl, command.PermissionExecute)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !allowed {
		return WorkflowRun{}, &ForbiddenError{Reason: reason}
	}

	if tpl.Status != TemplateStatusPublished {
		return WorkflowRun{}, ErrInvalidRequest
	}

	runVisibility := strings.TrimSpace(visibility)
	if runVisibility == "" {
		runVisibility = tpl.Visibility
	}
	runVisibility, err = s.normalizeVisibility(runVisibility)
	if err != nil {
		return WorkflowRun{}, err
	}

	return s.repo.CreateRun(ctx, CreateRunInput{
		Context:     req,
		TemplateID:  templateID,
		Visibility:  runVisibility,
		Inputs:      inputs,
		Mode:        normalizedMode,
		FromStepKey: fromStepKey,
		TestNode:    testNode,
		Now:         time.Now().UTC(),
	})
}

func (s *Service) RetryRun(
	ctx context.Context,
	req command.RequestContext,
	runID string,
	fromStepKey string,
	reason string,
	mode string,
) (WorkflowRun, error) {
	runID = strings.TrimSpace(runID)
	fromStepKey = strings.TrimSpace(fromStepKey)
	reason = strings.TrimSpace(reason)
	if runID == "" {
		return WorkflowRun{}, ErrInvalidRequest
	}

	normalizedMode, err := normalizeRetryMode(mode)
	if err != nil {
		return WorkflowRun{}, err
	}

	sourceRun, err := s.repo.GetRunForAccess(ctx, req, runID)
	if err != nil {
		return WorkflowRun{}, err
	}
	allowed, reasonCode, err := s.authorizeRun(ctx, req, sourceRun, command.PermissionExecute)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !allowed {
		return WorkflowRun{}, &ForbiddenError{Reason: reasonCode}
	}
	if sourceRun.Status == RunStatusPending || sourceRun.Status == RunStatusRunning {
		return WorkflowRun{}, ErrInvalidRequest
	}

	return s.repo.RetryRun(ctx, RetryRunInput{
		Context:     req,
		RunID:       runID,
		FromStepKey: fromStepKey,
		Reason:      reason,
		Mode:        normalizedMode,
		Now:         time.Now().UTC(),
	})
}

func (s *Service) CancelRun(ctx context.Context, req command.RequestContext, runID string) (WorkflowRun, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return WorkflowRun{}, ErrInvalidRequest
	}

	run, err := s.repo.GetRunForAccess(ctx, req, runID)
	if err != nil {
		return WorkflowRun{}, err
	}
	allowed, reason, err := s.authorizeRun(ctx, req, run, command.PermissionExecute)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !allowed {
		return WorkflowRun{}, &ForbiddenError{Reason: reason}
	}

	if run.Status == RunStatusSucceeded || run.Status == RunStatusFailed || run.Status == RunStatusCanceled {
		return run, nil
	}

	return s.repo.CancelRun(ctx, CancelRunInput{
		Context: req,
		RunID:   runID,
		Now:     time.Now().UTC(),
	})
}

func (s *Service) GetRun(ctx context.Context, req command.RequestContext, runID string) (WorkflowRun, error) {
	run, err := s.repo.GetRunForAccess(ctx, req, strings.TrimSpace(runID))
	if err != nil {
		return WorkflowRun{}, err
	}
	allowed, reason, err := s.authorizeRun(ctx, req, run, command.PermissionRead)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !allowed {
		return WorkflowRun{}, &ForbiddenError{Reason: reason}
	}
	return run, nil
}

func (s *Service) ListRuns(ctx context.Context, params RunListParams) (RunListResult, error) {
	return s.repo.ListRuns(ctx, params)
}

func (s *Service) ListStepRuns(ctx context.Context, params StepListParams) (StepListResult, error) {
	run, err := s.repo.GetRunForAccess(ctx, params.Context, params.RunID)
	if err != nil {
		return StepListResult{}, err
	}
	allowed, reason, err := s.authorizeRun(ctx, params.Context, run, command.PermissionRead)
	if err != nil {
		return StepListResult{}, err
	}
	if !allowed {
		return StepListResult{}, &ForbiddenError{Reason: reason}
	}
	return s.repo.ListStepRuns(ctx, params)
}

func (s *Service) ListRunEvents(ctx context.Context, req command.RequestContext, runID string) ([]WorkflowRunEvent, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, ErrInvalidRequest
	}

	run, err := s.repo.GetRunForAccess(ctx, req, runID)
	if err != nil {
		return nil, err
	}
	allowed, reason, err := s.authorizeRun(ctx, req, run, command.PermissionRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, &ForbiddenError{Reason: reason}
	}
	return s.repo.ListRunEvents(ctx, req, runID)
}

func (s *Service) authorizeTemplate(
	ctx context.Context,
	req command.RequestContext,
	tpl WorkflowTemplate,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != tpl.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != tpl.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == tpl.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && tpl.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasTemplatePermission(ctx, req, tpl.ID, permission, time.Now().UTC())
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

func (s *Service) authorizeRun(
	ctx context.Context,
	req command.RequestContext,
	run WorkflowRun,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != run.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != run.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == run.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && run.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasRunPermission(ctx, req, run.ID, permission, time.Now().UTC())
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

func normalizeRunMode(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", RunModeSync:
		return RunModeSync, nil
	case RunModeRunning:
		return RunModeRunning, nil
	case RunModeFail:
		return RunModeFail, nil
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeRetryMode(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", RunModeRetry:
		return RunModeRetry, nil
	default:
		return "", ErrInvalidRequest
	}
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}

func applyPatchOperationsToGraph(baseGraph json.RawMessage, opsRaw json.RawMessage) (json.RawMessage, error) {
	graph := map[string]any{}
	if len(baseGraph) > 0 {
		if err := json.Unmarshal(baseGraph, &graph); err != nil {
			return nil, err
		}
	}

	nodes := readObjectArray(graph["nodes"])
	edges := readObjectArray(graph["edges"])

	var operations []map[string]any
	if err := json.Unmarshal(opsRaw, &operations); err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, ErrInvalidRequest
	}

	for _, op := range operations {
		if err := applySinglePatchOperation(&nodes, &edges, op); err != nil {
			return nil, err
		}
	}

	graph["nodes"] = nodes
	graph["edges"] = edges
	return json.Marshal(graph)
}

func applySinglePatchOperation(nodes *[]map[string]any, edges *[]map[string]any, op map[string]any) error {
	opType := strings.ToLower(strings.TrimSpace(readPatchString(op, "op")))
	if opType == "" {
		return ErrInvalidRequest
	}
	path := strings.TrimSpace(readPatchString(op, "path"))
	value := readPatchObject(op, "value")

	switch opType {
	case "add_node":
		if value == nil {
			return ErrInvalidRequest
		}
		nodeID := strings.TrimSpace(readPatchString(value, "id"))
		if nodeID == "" || findNodeIndexByID(*nodes, nodeID) >= 0 {
			return ErrInvalidRequest
		}
		*nodes = append(*nodes, value)
		return nil
	case "update_node":
		nodeID := strings.TrimSpace(readPatchString(value, "id"))
		if nodeID == "" {
			nodeID = extractPatchPathID(path, "/nodes/")
		}
		if nodeID == "" || value == nil {
			return ErrInvalidRequest
		}
		idx := findNodeIndexByID(*nodes, nodeID)
		if idx < 0 {
			return ErrInvalidRequest
		}
		merged := mergePatchObject((*nodes)[idx], value)
		merged["id"] = nodeID
		(*nodes)[idx] = merged
		return nil
	case "remove_node":
		nodeID := strings.TrimSpace(readPatchString(value, "id"))
		if nodeID == "" {
			nodeID = extractPatchPathID(path, "/nodes/")
		}
		if nodeID == "" {
			return ErrInvalidRequest
		}
		idx := findNodeIndexByID(*nodes, nodeID)
		if idx < 0 {
			return ErrInvalidRequest
		}
		*nodes = append((*nodes)[:idx], (*nodes)[idx+1:]...)
		filteredEdges := make([]map[string]any, 0, len(*edges))
		for _, edge := range *edges {
			source := strings.TrimSpace(readPatchString(edge, "source"))
			if source == "" {
				source = strings.TrimSpace(readPatchString(edge, "from"))
			}
			target := strings.TrimSpace(readPatchString(edge, "target"))
			if target == "" {
				target = strings.TrimSpace(readPatchString(edge, "to"))
			}
			if source == nodeID || target == nodeID {
				continue
			}
			filteredEdges = append(filteredEdges, edge)
		}
		*edges = filteredEdges
		return nil
	case "add_edge":
		if value == nil {
			return ErrInvalidRequest
		}
		source := strings.TrimSpace(readPatchString(value, "source"))
		if source == "" {
			source = strings.TrimSpace(readPatchString(value, "from"))
		}
		target := strings.TrimSpace(readPatchString(value, "target"))
		if target == "" {
			target = strings.TrimSpace(readPatchString(value, "to"))
		}
		if source == "" || target == "" {
			return ErrInvalidRequest
		}
		if findNodeIndexByID(*nodes, source) < 0 || findNodeIndexByID(*nodes, target) < 0 {
			return ErrInvalidRequest
		}
		edgeID := strings.TrimSpace(readPatchString(value, "id"))
		if edgeID == "" {
			edgeID = fmt.Sprintf("e_%s_%s", source, target)
			value["id"] = edgeID
		}
		if findEdgeIndexByID(*edges, edgeID) >= 0 {
			return ErrInvalidRequest
		}
		*edges = append(*edges, value)
		return nil
	case "remove_edge":
		edgeID := strings.TrimSpace(readPatchString(value, "id"))
		if edgeID == "" {
			edgeID = extractPatchPathID(path, "/edges/")
		}
		if edgeID != "" {
			idx := findEdgeIndexByID(*edges, edgeID)
			if idx < 0 {
				return ErrInvalidRequest
			}
			*edges = append((*edges)[:idx], (*edges)[idx+1:]...)
			return nil
		}
		source := strings.TrimSpace(readPatchString(value, "source"))
		if source == "" {
			source = strings.TrimSpace(readPatchString(value, "from"))
		}
		target := strings.TrimSpace(readPatchString(value, "target"))
		if target == "" {
			target = strings.TrimSpace(readPatchString(value, "to"))
		}
		if source == "" || target == "" {
			return ErrInvalidRequest
		}
		idx := findEdgeIndexByEndpoints(*edges, source, target)
		if idx < 0 {
			return ErrInvalidRequest
		}
		*edges = append((*edges)[:idx], (*edges)[idx+1:]...)
		return nil
	case "annotate":
		// annotate updates metadata only; graph remains unchanged.
		return nil
	default:
		return ErrInvalidRequest
	}
}

func readObjectArray(raw any) []map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, obj)
	}
	return out
}

func readPatchString(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	value, _ := raw[key].(string)
	return strings.TrimSpace(value)
}

func readPatchObject(raw map[string]any, key string) map[string]any {
	if raw == nil {
		return nil
	}
	value, _ := raw[key].(map[string]any)
	return value
}

func extractPatchPathID(path string, prefix string) string {
	normalized := strings.TrimSpace(path)
	if normalized == "" || !strings.HasPrefix(normalized, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(normalized, prefix), "/")
}

func findNodeIndexByID(nodes []map[string]any, nodeID string) int {
	target := strings.TrimSpace(nodeID)
	for idx, node := range nodes {
		if strings.TrimSpace(readPatchString(node, "id")) == target {
			return idx
		}
	}
	return -1
}

func findEdgeIndexByID(edges []map[string]any, edgeID string) int {
	target := strings.TrimSpace(edgeID)
	for idx, edge := range edges {
		if strings.TrimSpace(readPatchString(edge, "id")) == target {
			return idx
		}
	}
	return -1
}

func findEdgeIndexByEndpoints(edges []map[string]any, source string, target string) int {
	src := strings.TrimSpace(source)
	dst := strings.TrimSpace(target)
	for idx, edge := range edges {
		edgeSource := strings.TrimSpace(readPatchString(edge, "source"))
		if edgeSource == "" {
			edgeSource = strings.TrimSpace(readPatchString(edge, "from"))
		}
		edgeTarget := strings.TrimSpace(readPatchString(edge, "target"))
		if edgeTarget == "" {
			edgeTarget = strings.TrimSpace(readPatchString(edge, "to"))
		}
		if edgeSource == src && edgeTarget == dst {
			return idx
		}
	}
	return -1
}

func mergePatchObject(base map[string]any, patch map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(patch))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range patch {
		merged[key] = value
	}
	return merged
}
