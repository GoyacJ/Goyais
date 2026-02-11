// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"goyais/internal/ai"
	"goyais/internal/algorithm"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/contextbundle"
	"goyais/internal/plugin"
	"goyais/internal/stream"
	"goyais/internal/workflow"
)

type assetUploadCommandPayload struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Mime       string `json:"mime"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	Visibility string `json:"visibility"`
	FileBase64 string `json:"fileBase64"`
}

type assetUpdateCommandPayload struct {
	AssetID    string           `json:"assetId"`
	Name       *string          `json:"name"`
	Visibility *string          `json:"visibility"`
	Metadata   *json.RawMessage `json:"metadata"`
}

type assetDeleteCommandPayload struct {
	AssetID string `json:"assetId"`
}

type workflowTemplateCreateCommandPayload struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Graph         json.RawMessage `json:"graph"`
	SchemaInputs  json.RawMessage `json:"schemaInputs"`
	SchemaOutputs json.RawMessage `json:"schemaOutputs"`
	UIState       json.RawMessage `json:"uiState"`
	Visibility    string          `json:"visibility"`
}

type workflowTemplatePatchCommandPayload struct {
	TemplateID string          `json:"templateId"`
	Patch      json.RawMessage `json:"patch"`
}

type workflowTemplatePublishCommandPayload struct {
	TemplateID string `json:"templateId"`
}

type workflowRunCommandPayload struct {
	TemplateID  string          `json:"templateId"`
	Inputs      json.RawMessage `json:"inputs"`
	Visibility  string          `json:"visibility"`
	Mode        string          `json:"mode"`
	FromStepKey string          `json:"fromStepKey"`
	TestNode    bool            `json:"testNode"`
}

type workflowRetryCommandPayload struct {
	RunID       string `json:"runId"`
	FromStepKey string `json:"fromStepKey"`
	Reason      string `json:"reason"`
	Mode        string `json:"mode"`
}

type workflowCancelCommandPayload struct {
	RunID string `json:"runId"`
}

type algorithmRunCommandPayload struct {
	AlgorithmID string          `json:"algorithmId"`
	Inputs      json.RawMessage `json:"inputs"`
	Visibility  string          `json:"visibility"`
	Mode        string          `json:"mode"`
}

type shareCreateCommandPayload struct {
	ResourceType string   `json:"resourceType"`
	ResourceID   string   `json:"resourceId"`
	SubjectType  string   `json:"subjectType"`
	SubjectID    string   `json:"subjectId"`
	Permissions  []string `json:"permissions"`
	ExpiresAt    string   `json:"expiresAt"`
}

type shareDeleteCommandPayload struct {
	ShareID string `json:"shareId"`
}

type pluginUploadCommandPayload struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	PackageType string          `json:"packageType"`
	Manifest    json.RawMessage `json:"manifest"`
	Visibility  string          `json:"visibility"`
}

type pluginInstallCommandPayload struct {
	PackageID string `json:"packageId"`
	Scope     string `json:"scope"`
}

type pluginInstallActionCommandPayload struct {
	InstallID string `json:"installId"`
}

type streamCreateCommandPayload struct {
	Path       string          `json:"path"`
	Protocol   string          `json:"protocol"`
	Source     string          `json:"source"`
	Visibility string          `json:"visibility"`
	State      json.RawMessage `json:"state"`
}

type streamActionCommandPayload struct {
	StreamID string `json:"streamId"`
}

type streamUpdateAuthCommandPayload struct {
	StreamID string          `json:"streamId"`
	AuthRule json.RawMessage `json:"authRule"`
}

type aiSessionCreateCommandPayload struct {
	Title       string          `json:"title"`
	Goal        string          `json:"goal"`
	Visibility  string          `json:"visibility"`
	Inputs      json.RawMessage `json:"inputs"`
	Constraints json.RawMessage `json:"constraints"`
	Preferences json.RawMessage `json:"preferences"`
}

type aiSessionArchiveCommandPayload struct {
	SessionID string `json:"sessionId"`
}

type aiSessionTurnCommandPayload struct {
	SessionID          string          `json:"sessionId"`
	Message            string          `json:"message"`
	Execute            bool            `json:"execute"`
	IntentCommandType  string          `json:"intentCommandType"`
	IntentCommandInput json.RawMessage `json:"intentPayload"`
}

type contextBundleRebuildCommandPayload struct {
	ScopeType  string `json:"scopeType"`
	ScopeID    string `json:"scopeId"`
	Visibility string `json:"visibility"`
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func registerCommandExecutors(
	commandService *command.Service,
	aiService *ai.Service,
	aiWorkbenchEnabled bool,
	assetService *asset.Service,
	assetLifecycleEnabled bool,
	pluginService *plugin.Service,
	workflowService *workflow.Service,
	streamService *stream.Service,
	streamControlPlaneEnabled bool,
	algorithmService *algorithm.Service,
	contextBundleService *contextbundle.Service,
	contextBundleEnabled bool,
) {
	if commandService == nil {
		return
	}
	commandService.SetExecutor("plugin.upgrade", newNotImplementedExecutor("error.plugin.not_implemented"))
	commandService.SetExecutor("context.bundle.rebuild", newNotImplementedExecutor("error.context_bundle.not_implemented"))
	commandService.SetExecutor("stream.updateAuth", newNotImplementedExecutor("error.stream.not_implemented"))
	commandService.SetExecutor("stream.delete", newNotImplementedExecutor("error.stream.not_implemented"))
	commandService.SetExecutor("share.create", newShareCreateExecutor(commandService))
	commandService.SetExecutor("share.delete", newShareDeleteExecutor(commandService))
	commandService.SetExecutor("ai.session.create", newNotImplementedExecutor("error.ai.not_implemented"))
	commandService.SetExecutor("ai.session.archive", newNotImplementedExecutor("error.ai.not_implemented"))
	commandService.SetExecutor("ai.intent.plan", newNotImplementedExecutor("error.ai.not_implemented"))
	commandService.SetExecutor("ai.command.execute", newNotImplementedExecutor("error.ai.not_implemented"))
	if aiService != nil && aiWorkbenchEnabled {
		commandService.SetExecutor("ai.session.create", newAISessionCreateExecutor(aiService))
		commandService.SetExecutor("ai.session.archive", newAISessionArchiveExecutor(aiService))
		commandService.SetExecutor("ai.intent.plan", newAISessionTurnExecutor(aiService, commandService, "ai.intent.plan"))
		commandService.SetExecutor("ai.command.execute", newAISessionTurnExecutor(aiService, commandService, "ai.command.execute"))
	}
	if assetService != nil {
		commandService.SetExecutor("asset.upload", newAssetUploadExecutor(assetService))
		if assetLifecycleEnabled {
			commandService.SetExecutor("asset.update", newAssetUpdateExecutor(assetService))
			commandService.SetExecutor("asset.delete", newAssetDeleteExecutor(assetService))
		}
	}
	if workflowService != nil {
		commandService.SetExecutor("workflow.createDraft", newWorkflowCreateDraftExecutor(workflowService))
		commandService.SetExecutor("workflow.patch", newWorkflowPatchExecutor(workflowService))
		commandService.SetExecutor("workflow.publish", newWorkflowPublishExecutor(workflowService))
		commandService.SetExecutor("workflow.run", newWorkflowRunExecutor(workflowService))
		commandService.SetExecutor("workflow.retry", newWorkflowRetryExecutor(workflowService))
		commandService.SetExecutor("workflow.cancel", newWorkflowCancelExecutor(workflowService))
	}
	if pluginService != nil {
		commandService.SetExecutor("plugin.upload", newPluginUploadExecutor(pluginService))
		commandService.SetExecutor("plugin.install", newPluginInstallExecutor(pluginService))
		commandService.SetExecutor("plugin.enable", newPluginEnableExecutor(pluginService))
		commandService.SetExecutor("plugin.disable", newPluginDisableExecutor(pluginService))
		commandService.SetExecutor("plugin.rollback", newPluginRollbackExecutor(pluginService))
		commandService.SetExecutor("plugin.upgrade", newPluginUpgradeExecutor(pluginService))
	}
	if streamService != nil {
		commandService.SetExecutor("stream.create", newStreamCreateExecutor(streamService))
		commandService.SetExecutor("stream.record.start", newStreamRecordStartExecutor(commandService, streamService))
		commandService.SetExecutor("stream.record.stop", newStreamRecordStopExecutor(commandService, streamService))
		commandService.SetExecutor("stream.kick", newStreamKickExecutor(streamService))
		if streamControlPlaneEnabled {
			commandService.SetExecutor("stream.updateAuth", newStreamUpdateAuthExecutor(streamService))
			commandService.SetExecutor("stream.delete", newStreamDeleteExecutor(streamService))
		}
	}
	if algorithmService != nil {
		commandService.SetExecutor("algorithm.run", newAlgorithmRunExecutor(algorithmService))
	}
	if contextBundleService != nil && contextBundleEnabled {
		commandService.SetExecutor("context.bundle.rebuild", newContextBundleRebuildExecutor(contextBundleService))
	}
}

func newNotImplementedExecutor(messageKey string) command.ExecuteFunc {
	return func(_ context.Context, _ command.RequestContext, _ json.RawMessage) ([]byte, error) {
		return nil, &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: messageKey,
			Err:        command.ErrNotImplemented,
		}
	}
}

func newAssetUploadExecutor(assetService *asset.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req assetUploadCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		fileBase64 := strings.TrimSpace(req.FileBase64)
		if fileBase64 == "" {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}
		fileData, err := base64.StdEncoding.DecodeString(fileBase64)
		if err != nil || len(fileData) == 0 {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		hash := sha256.Sum256(fileData)
		computedHash := hex.EncodeToString(hash[:])
		requestHash := strings.ToLower(strings.TrimSpace(req.Hash))
		if requestHash != "" && requestHash != computedHash {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		mimeType := strings.TrimSpace(req.Mime)
		if mimeType == "" {
			mimeType = http.DetectContentType(fileData)
		}

		created, err := assetService.Create(ctx, asset.CreateInput{
			Context:    reqCtx,
			Name:       name,
			Type:       strings.TrimSpace(req.Type),
			Mime:       mimeType,
			Size:       int64(len(fileData)),
			Hash:       computedHash,
			Visibility: strings.TrimSpace(req.Visibility),
			Now:        time.Now().UTC(),
		}, fileData)
		if err != nil {
			return nil, mapAssetExecutionError(err)
		}

		result := map[string]any{
			"asset": toAssetResultPayload(created),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newAssetUpdateExecutor(assetService *asset.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req assetUpdateCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}
		assetID := strings.TrimSpace(req.AssetID)
		if assetID == "" {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		updateInput := asset.UpdateInput{
			Context: reqCtx,
			AssetID: assetID,
			Name:    req.Name,
			Now:     time.Now().UTC(),
		}
		if req.Visibility != nil {
			updateInput.Visibility = req.Visibility
		}
		if req.Metadata != nil {
			rawMetadata := strings.TrimSpace(string(*req.Metadata))
			if rawMetadata == "" || rawMetadata == "null" {
				updateInput.Metadata = json.RawMessage(`{}`)
			} else {
				var metadataObj map[string]any
				if err := json.Unmarshal(*req.Metadata, &metadataObj); err != nil {
					return nil, &command.ExecutionError{
						Code:       "INVALID_ASSET_REQUEST",
						MessageKey: "error.asset.invalid_request",
						Err:        command.ErrInvalidCommandRequest,
					}
				}
				updateInput.Metadata = *req.Metadata
			}
			updateInput.MetadataSet = true
		}

		updated, err := assetService.Update(ctx, updateInput)
		if err != nil {
			return nil, mapAssetExecutionError(err)
		}
		result := map[string]any{
			"asset": toAssetResultPayload(updated),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newAssetDeleteExecutor(assetService *asset.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req assetDeleteCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}
		assetID := strings.TrimSpace(req.AssetID)
		if assetID == "" {
			return nil, &command.ExecutionError{
				Code:       "INVALID_ASSET_REQUEST",
				MessageKey: "error.asset.invalid_request",
				Err:        command.ErrInvalidCommandRequest,
			}
		}

		deleted, err := assetService.Delete(ctx, reqCtx, assetID)
		if err != nil {
			return nil, mapAssetExecutionError(err)
		}
		result := map[string]any{
			"asset": toAssetResultPayload(deleted),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapAssetExecutionError(err error) error {
	switch {
	case errors.Is(err, asset.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_ASSET_REQUEST",
			MessageKey: "error.asset.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, asset.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.asset.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, asset.ErrNotFound):
		return &command.ExecutionError{
			Code:       "ASSET_NOT_FOUND",
			MessageKey: "error.asset.not_found",
			Err:        command.ErrNotFound,
		}
	case errors.Is(err, asset.ErrForbidden):
		reason := ""
		var forbidden *asset.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("asset executor: %w", err),
		}
	}
}

func newAISessionCreateExecutor(aiService *ai.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req aiSessionCreateCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapAIExecutionError(ai.ErrInvalidRequest)
		}
		if len(req.Inputs) == 0 {
			req.Inputs = json.RawMessage(`{}`)
		}
		if len(req.Constraints) == 0 {
			req.Constraints = json.RawMessage(`{}`)
		}
		if len(req.Preferences) == 0 {
			req.Preferences = json.RawMessage(`{}`)
		}

		session, err := aiService.CreateSession(
			ctx,
			reqCtx,
			req.Title,
			req.Goal,
			objectOrDefault(req.Inputs),
			objectOrDefault(req.Constraints),
			objectOrDefault(req.Preferences),
			req.Visibility,
		)
		if err != nil {
			return nil, mapAIExecutionError(err)
		}

		result := map[string]any{
			"session": toAISessionResultPayload(session),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newAISessionArchiveExecutor(aiService *ai.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req aiSessionArchiveCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapAIExecutionError(ai.ErrInvalidRequest)
		}

		session, err := aiService.ArchiveSession(ctx, reqCtx, req.SessionID)
		if err != nil {
			return nil, mapAIExecutionError(err)
		}

		result := map[string]any{
			"session": toAISessionResultPayload(session),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newAISessionTurnExecutor(aiService *ai.Service, commandService *command.Service, commandType string) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req aiSessionTurnCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapAIExecutionError(ai.ErrInvalidRequest)
		}
		assistantMessage := ""
		commandIDs := make([]string, 0, 2)
		plan, err := planAIDeterministicCommand(req)
		if err != nil {
			return nil, mapAIExecutionError(ai.ErrInvalidRequest)
		}

		if commandType == "ai.command.execute" {
			if strings.TrimSpace(plan.CommandType) == "" {
				assistantMessage = "No executable intent detected. Use `run workflow <templateId>` or provide `intentCommandType`."
			} else {
				if commandService == nil {
					return nil, mapAIExecutionError(ai.ErrNotImplemented)
				}
				child, err := commandService.Submit(ctx, reqCtx, plan.CommandType, plan.Payload, "", "")
				if err != nil {
					return nil, err
				}
				commandIDs = append(commandIDs, child.ID)
				assistantMessage = buildAIExecuteAssistantMessage(plan.CommandType, child)
			}
		} else if strings.TrimSpace(plan.CommandType) != "" {
			assistantMessage = buildAIPlanAssistantMessage(plan)
		}

		turn, err := aiService.CreateTurn(ctx, reqCtx, req.SessionID, req.Message, commandType, assistantMessage, commandIDs)
		if err != nil {
			return nil, mapAIExecutionError(err)
		}

		result := map[string]any{
			"turn": toAISessionTurnResultPayload(turn),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

type aiDeterministicCommandPlan struct {
	CommandType string
	Payload     json.RawMessage
}

func planAIDeterministicCommand(req aiSessionTurnCommandPayload) (aiDeterministicCommandPlan, error) {
	explicitType := strings.TrimSpace(req.IntentCommandType)
	if explicitType != "" {
		if strings.HasPrefix(strings.ToLower(explicitType), "ai.") {
			return aiDeterministicCommandPlan{}, ai.ErrInvalidRequest
		}
		payload := objectOrDefault(req.IntentCommandInput)
		if !isJSONObjectRaw(payload) {
			return aiDeterministicCommandPlan{}, ai.ErrInvalidRequest
		}
		return aiDeterministicCommandPlan{
			CommandType: explicitType,
			Payload:     payload,
		}, nil
	}

	tokens := strings.Fields(strings.TrimSpace(req.Message))
	switch {
	case len(tokens) >= 3 && strings.EqualFold(tokens[0], "run") && strings.EqualFold(tokens[1], "workflow"):
		return buildWorkflowRunPlan(cleanAICommandToken(tokens[2])), nil
	case len(tokens) >= 2 && strings.EqualFold(tokens[0], "workflow.run"):
		return buildWorkflowRunPlan(cleanAICommandToken(tokens[1])), nil
	case len(tokens) >= 3 && strings.EqualFold(tokens[0], "retry") && strings.EqualFold(tokens[1], "workflow"):
		return buildWorkflowRetryPlan(cleanAICommandToken(tokens[2])), nil
	case len(tokens) >= 2 && strings.EqualFold(tokens[0], "workflow.retry"):
		return buildWorkflowRetryPlan(cleanAICommandToken(tokens[1])), nil
	case len(tokens) >= 3 && strings.EqualFold(tokens[0], "cancel") && strings.EqualFold(tokens[1], "workflow"):
		return buildWorkflowCancelPlan(cleanAICommandToken(tokens[2])), nil
	case len(tokens) >= 2 && strings.EqualFold(tokens[0], "workflow.cancel"):
		return buildWorkflowCancelPlan(cleanAICommandToken(tokens[1])), nil
	default:
		return aiDeterministicCommandPlan{}, nil
	}
}

func buildWorkflowRunPlan(templateID string) aiDeterministicCommandPlan {
	if templateID == "" {
		return aiDeterministicCommandPlan{}
	}
	payload, _ := json.Marshal(map[string]any{
		"templateId": templateID,
		"mode":       "sync",
		"inputs":     map[string]any{},
	})
	return aiDeterministicCommandPlan{
		CommandType: "workflow.run",
		Payload:     payload,
	}
}

func buildWorkflowRetryPlan(runID string) aiDeterministicCommandPlan {
	if runID == "" {
		return aiDeterministicCommandPlan{}
	}
	payload, _ := json.Marshal(map[string]any{
		"runId": runID,
		"mode":  "sync",
	})
	return aiDeterministicCommandPlan{
		CommandType: "workflow.retry",
		Payload:     payload,
	}
}

func buildWorkflowCancelPlan(runID string) aiDeterministicCommandPlan {
	if runID == "" {
		return aiDeterministicCommandPlan{}
	}
	payload, _ := json.Marshal(map[string]any{
		"runId": runID,
	})
	return aiDeterministicCommandPlan{
		CommandType: "workflow.cancel",
		Payload:     payload,
	}
}

func cleanAICommandToken(raw string) string {
	token := strings.TrimSpace(raw)
	token = strings.Trim(token, "`\"'.,;:()[]{}")
	return token
}

func isJSONObjectRaw(raw json.RawMessage) bool {
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}

func buildAIPlanAssistantMessage(plan aiDeterministicCommandPlan) string {
	if strings.TrimSpace(plan.CommandType) == "" {
		return ""
	}
	return fmt.Sprintf("Planned command %s with payload %s", plan.CommandType, string(plan.Payload))
}

func buildAIExecuteAssistantMessage(commandType string, cmd command.Command) string {
	if runID := extractWorkflowRunID(cmd.Result); runID != "" {
		return fmt.Sprintf("Executed %s via command %s (workflowRun=%s, status=%s).", commandType, cmd.ID, runID, cmd.Status)
	}
	return fmt.Sprintf("Executed %s via command %s (status=%s).", commandType, cmd.ID, cmd.Status)
}

func extractWorkflowRunID(resultRaw json.RawMessage) string {
	if len(resultRaw) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(resultRaw, &payload); err != nil {
		return ""
	}
	runRaw, ok := payload["run"].(map[string]any)
	if !ok {
		return ""
	}
	runID, _ := runRaw["id"].(string)
	return strings.TrimSpace(runID)
}

func mapAIExecutionError(err error) error {
	switch {
	case errors.Is(err, ai.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_AI_REQUEST",
			MessageKey: "error.ai.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, ai.ErrSessionNotFound):
		return &command.ExecutionError{
			Code:       "AI_SESSION_NOT_FOUND",
			MessageKey: "error.ai.not_found",
			Err:        command.ErrNotFound,
		}
	case errors.Is(err, ai.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.ai.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, ai.ErrForbidden):
		reason := ""
		var forbidden *ai.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("ai executor: %w", err),
		}
	}
}

func newWorkflowCreateDraftExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowTemplateCreateCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}
		template, err := workflowService.CreateTemplateDraft(
			ctx,
			reqCtx,
			req.Name,
			req.Description,
			objectOrDefault(req.Graph),
			objectOrDefault(req.SchemaInputs),
			objectOrDefault(req.SchemaOutputs),
			objectOrDefault(req.UIState),
			req.Visibility,
		)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"template": toWorkflowTemplateResultPayload(template),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newWorkflowPatchExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowTemplatePatchCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}

		template, err := workflowService.PatchTemplate(
			ctx,
			reqCtx,
			req.TemplateID,
			req.Patch,
		)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"template": toWorkflowTemplateResultPayload(template),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newWorkflowPublishExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowTemplatePublishCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}

		template, err := workflowService.PublishTemplate(ctx, reqCtx, req.TemplateID)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"template": toWorkflowTemplateResultPayload(template),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newWorkflowRunExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowRunCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}

		run, err := workflowService.CreateRun(
			ctx,
			reqCtx,
			req.TemplateID,
			objectOrDefault(req.Inputs),
			req.Visibility,
			req.Mode,
			req.FromStepKey,
			req.TestNode,
		)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"run": toWorkflowRunResultPayload(run),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newWorkflowCancelExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowCancelCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}

		run, err := workflowService.CancelRun(ctx, reqCtx, req.RunID)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"run": toWorkflowRunResultPayload(run),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newWorkflowRetryExecutor(workflowService *workflow.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req workflowRetryCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapWorkflowExecutionError(workflow.ErrInvalidRequest)
		}

		run, err := workflowService.RetryRun(
			ctx,
			reqCtx,
			req.RunID,
			req.FromStepKey,
			req.Reason,
			req.Mode,
		)
		if err != nil {
			return nil, mapWorkflowExecutionError(err)
		}

		result := map[string]any{
			"run": toWorkflowRunResultPayload(run),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapWorkflowExecutionError(err error) error {
	switch {
	case errors.Is(err, workflow.ErrInvalidRequest), errors.Is(err, workflow.ErrTemplateNotFound), errors.Is(err, workflow.ErrRunNotFound):
		return &command.ExecutionError{
			Code:       "INVALID_WORKFLOW_REQUEST",
			MessageKey: "error.workflow.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, workflow.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.workflow.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, workflow.ErrForbidden):
		reason := ""
		var forbidden *workflow.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("workflow executor: %w", err),
		}
	}
}

func newAlgorithmRunExecutor(algorithmService *algorithm.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req algorithmRunCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapAlgorithmExecutionError(algorithm.ErrInvalidRequest)
		}
		run, err := algorithmService.Run(ctx, algorithm.RunInput{
			Context:     reqCtx,
			AlgorithmID: req.AlgorithmID,
			Inputs:      objectOrDefault(req.Inputs),
			Visibility:  req.Visibility,
			Mode:        req.Mode,
		})
		if err != nil {
			return nil, mapAlgorithmExecutionError(err)
		}

		result := map[string]any{
			"run": toAlgorithmRunResultPayload(run),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapAlgorithmExecutionError(err error) error {
	switch {
	case errors.Is(err, algorithm.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_ALGORITHM_REQUEST",
			MessageKey: "error.algorithm.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, algorithm.ErrAlgorithmNotFound):
		return &command.ExecutionError{
			Code:       "ALGORITHM_NOT_FOUND",
			MessageKey: "error.algorithm.not_found",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, algorithm.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.algorithm.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, algorithm.ErrForbidden):
		reason := ""
		var forbidden *algorithm.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("algorithm executor: %w", err),
		}
	}
}

func newPluginUploadExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginUploadCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}
		if len(req.Manifest) == 0 {
			req.Manifest = json.RawMessage(`{}`)
		}

		pkg, err := pluginService.UploadPackage(
			ctx,
			reqCtx,
			req.Name,
			req.Version,
			req.PackageType,
			req.Manifest,
			req.Visibility,
		)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"package": toPluginPackageResultPayload(pkg),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newPluginInstallExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginInstallCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}

		ins, err := pluginService.InstallPackage(ctx, reqCtx, req.PackageID, req.Scope)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"install": toPluginInstallResultPayload(ins),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newPluginEnableExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginInstallActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}

		ins, err := pluginService.EnableInstall(ctx, reqCtx, req.InstallID)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"install": toPluginInstallResultPayload(ins),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newPluginDisableExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginInstallActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}

		ins, err := pluginService.DisableInstall(ctx, reqCtx, req.InstallID)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"install": toPluginInstallResultPayload(ins),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newPluginRollbackExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginInstallActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}

		ins, err := pluginService.RollbackInstall(ctx, reqCtx, req.InstallID)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"install": toPluginInstallResultPayload(ins),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newPluginUpgradeExecutor(pluginService *plugin.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req pluginInstallActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapPluginExecutionError(plugin.ErrInvalidRequest)
		}

		ins, err := pluginService.UpgradeInstall(ctx, reqCtx, req.InstallID)
		if err != nil {
			return nil, mapPluginExecutionError(err)
		}

		result := map[string]any{
			"install": toPluginInstallResultPayload(ins),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapPluginExecutionError(err error) error {
	switch {
	case errors.Is(err, plugin.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_PLUGIN_REQUEST",
			MessageKey: "error.plugin.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, plugin.ErrPackageNotFound), errors.Is(err, plugin.ErrInstallNotFound):
		return &command.ExecutionError{
			Code:       "PLUGIN_NOT_FOUND",
			MessageKey: "error.plugin.not_found",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, plugin.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.plugin.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, plugin.ErrForbidden):
		reason := ""
		var forbidden *plugin.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("plugin executor: %w", err),
		}
	}
}

func newStreamCreateExecutor(streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamCreateCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}
		if len(req.State) == 0 {
			req.State = json.RawMessage(`{}`)
		}

		created, err := streamService.CreateStream(
			ctx,
			reqCtx,
			req.Path,
			req.Protocol,
			req.Source,
			req.Visibility,
			req.State,
		)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}

		result := map[string]any{
			"stream": toStreamResultPayload(created),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newStreamRecordStartExecutor(commandService *command.Service, streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}

		started, err := streamService.StartRecording(ctx, reqCtx, req.StreamID)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}

		result := map[string]any{
			"stream":    toStreamResultPayload(started.Stream),
			"recording": toStreamRecordingResultPayload(started.Recording),
		}
		if status := strings.TrimSpace(started.OnPublishEventStatus); status != "" {
			eventBus := map[string]any{
				"status": status,
			}
			if errText := strings.TrimSpace(started.OnPublishEventError); errText != "" {
				eventBus["error"] = errText
			}
			result["eventBus"] = eventBus
		}

		if templateID := strings.TrimSpace(started.OnPublishTemplateID); templateID != "" && commandService != nil {
			workflowPayload, _ := json.Marshal(map[string]any{
				"templateId": templateID,
				"inputs": map[string]any{
					"streamId":    started.Stream.ID,
					"recordingId": started.Recording.ID,
					"trigger":     "stream.onPublish",
				},
				"visibility": started.Stream.Visibility,
				"mode":       "sync",
			})
			onPublish := map[string]any{
				"templateId": templateID,
			}
			onPublishCommand, submitErr := commandService.Submit(
				ctx,
				reqCtx,
				"workflow.run",
				workflowPayload,
				"stream-onpublish-"+started.Recording.ID,
				started.Stream.Visibility,
			)
			if submitErr != nil {
				onPublish["status"] = "failed"
				onPublish["error"] = submitErr.Error()
			} else {
				onPublish["status"] = "submitted"
				onPublish["commandId"] = onPublishCommand.ID
			}
			result["onPublish"] = onPublish
		}

		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newStreamRecordStopExecutor(commandService *command.Service, streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}

		stopped, err := streamService.StopRecording(ctx, reqCtx, req.StreamID)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}

		result := map[string]any{
			"stream":    toStreamResultPayload(stopped.Stream),
			"recording": toStreamRecordingResultPayload(stopped.Recording),
			"assetId":   stopped.AssetID,
			"lineageId": stopped.LineageID,
		}
		if status := strings.TrimSpace(stopped.OnRecordFinishEventStatus); status != "" {
			eventBus := map[string]any{
				"status": status,
			}
			if errText := strings.TrimSpace(stopped.OnRecordFinishEventError); errText != "" {
				eventBus["error"] = errText
			}
			result["eventBus"] = eventBus
		}
		if templateID := strings.TrimSpace(stopped.OnRecordFinishTemplateID); templateID != "" && commandService != nil {
			workflowPayload, _ := json.Marshal(map[string]any{
				"templateId": templateID,
				"inputs": map[string]any{
					"streamId":    stopped.Stream.ID,
					"recordingId": stopped.Recording.ID,
					"assetId":     stopped.AssetID,
					"trigger":     "stream.onRecordFinish",
				},
				"visibility": stopped.Stream.Visibility,
				"mode":       "sync",
			})
			onRecordFinish := map[string]any{
				"templateId": templateID,
			}
			onRecordFinishCommand, submitErr := commandService.Submit(
				ctx,
				reqCtx,
				"workflow.run",
				workflowPayload,
				"stream-onrecordfinish-"+stopped.Recording.ID,
				stopped.Stream.Visibility,
			)
			if submitErr != nil {
				onRecordFinish["status"] = "failed"
				onRecordFinish["error"] = submitErr.Error()
			} else {
				onRecordFinish["status"] = "submitted"
				onRecordFinish["commandId"] = onRecordFinishCommand.ID
			}
			result["onRecordFinish"] = onRecordFinish
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newStreamKickExecutor(streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}

		item, err := streamService.KickStream(ctx, reqCtx, req.StreamID)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}

		result := map[string]any{
			"stream": toStreamResultPayload(item),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newStreamUpdateAuthExecutor(streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamUpdateAuthCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}
		if len(req.AuthRule) == 0 {
			req.AuthRule = json.RawMessage(`{}`)
		}

		item, err := streamService.UpdateAuthRule(ctx, reqCtx, req.StreamID, req.AuthRule)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}
		result := map[string]any{
			"stream": toStreamResultPayload(item),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newStreamDeleteExecutor(streamService *stream.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req streamActionCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapStreamExecutionError(stream.ErrInvalidRequest)
		}

		item, err := streamService.DeleteStream(ctx, reqCtx, req.StreamID)
		if err != nil {
			return nil, mapStreamExecutionError(err)
		}
		result := map[string]any{
			"stream": toStreamResultPayload(item),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapStreamExecutionError(err error) error {
	switch {
	case errors.Is(err, stream.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_STREAM_REQUEST",
			MessageKey: "error.stream.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, stream.ErrStreamNotFound), errors.Is(err, stream.ErrRecordingNotFound):
		return &command.ExecutionError{
			Code:       "STREAM_NOT_FOUND",
			MessageKey: "error.stream.not_found",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, stream.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.stream.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, stream.ErrForbidden):
		reason := ""
		var forbidden *stream.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("stream executor: %w", err),
		}
	}
}

func newContextBundleRebuildExecutor(contextBundleService *contextbundle.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req contextBundleRebuildCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapContextBundleExecutionError(contextbundle.ErrInvalidRequest)
		}
		bundle, err := contextBundleService.RebuildBundle(ctx, reqCtx, req.ScopeType, req.ScopeID, req.Visibility)
		if err != nil {
			return nil, mapContextBundleExecutionError(err)
		}
		result := map[string]any{
			"bundle": map[string]any{
				"id":         bundle.ID,
				"status":     command.StatusSucceeded,
				"scopeType":  bundle.ScopeType,
				"scopeId":    bundle.ScopeID,
				"visibility": bundle.Visibility,
			},
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapContextBundleExecutionError(err error) error {
	switch {
	case errors.Is(err, contextbundle.ErrInvalidRequest):
		return &command.ExecutionError{
			Code:       "INVALID_CONTEXT_BUNDLE_REQUEST",
			MessageKey: "error.context_bundle.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, contextbundle.ErrNotFound):
		return &command.ExecutionError{
			Code:       "CONTEXT_BUNDLE_NOT_FOUND",
			MessageKey: "error.context_bundle.not_found",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, contextbundle.ErrForbidden):
		reason := ""
		var forbidden *contextbundle.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("context bundle executor: %w", err),
		}
	}
}

func newShareCreateExecutor(commandService *command.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req shareCreateCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapShareExecutionError(command.ErrInvalidShareRequest)
		}

		var expiresAt *time.Time
		if rawExpires := strings.TrimSpace(req.ExpiresAt); rawExpires != "" {
			parsed, err := time.Parse(timeRFC3339Nano, rawExpires)
			if err != nil {
				return nil, mapShareExecutionError(command.ErrInvalidShareRequest)
			}
			expiresAt = &parsed
		}

		created, err := commandService.CreateShare(
			ctx,
			reqCtx,
			req.ResourceType,
			req.ResourceID,
			req.SubjectType,
			req.SubjectID,
			req.Permissions,
			expiresAt,
		)
		if err != nil {
			return nil, mapShareExecutionError(err)
		}

		result := map[string]any{
			"share": toShareResultPayload(created),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func newShareDeleteExecutor(commandService *command.Service) command.ExecuteFunc {
	return func(ctx context.Context, reqCtx command.RequestContext, payload json.RawMessage) ([]byte, error) {
		var req shareDeleteCommandPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, mapShareExecutionError(command.ErrInvalidShareRequest)
		}
		shareID := strings.TrimSpace(req.ShareID)
		if shareID == "" {
			return nil, mapShareExecutionError(command.ErrInvalidShareRequest)
		}

		if err := commandService.DeleteShare(ctx, reqCtx, shareID); err != nil {
			return nil, mapShareExecutionError(err)
		}

		result := map[string]any{
			"share": toShareDeleteResultPayload(shareID),
		}
		raw, _ := json.Marshal(result)
		return raw, nil
	}
}

func mapShareExecutionError(err error) error {
	switch {
	case errors.Is(err, command.ErrInvalidShareRequest):
		return &command.ExecutionError{
			Code:       "INVALID_SHARE_REQUEST",
			MessageKey: "error.share.invalid_request",
			Err:        command.ErrInvalidCommandRequest,
		}
	case errors.Is(err, command.ErrNotImplemented):
		return &command.ExecutionError{
			Code:       "NOT_IMPLEMENTED",
			MessageKey: "error.share.not_implemented",
			Err:        command.ErrNotImplemented,
		}
	case errors.Is(err, command.ErrShareNotFound):
		return &command.ExecutionError{
			Code:       "SHARE_NOT_FOUND",
			MessageKey: "error.share.not_found",
			Err:        command.ErrShareNotFound,
		}
	case errors.Is(err, command.ErrForbidden):
		reason := ""
		var forbidden *command.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &command.ExecutionError{
			Code:       "FORBIDDEN",
			MessageKey: "error.authz.forbidden",
			Err:        &command.ForbiddenError{Reason: reason},
		}
	default:
		return &command.ExecutionError{
			Code:       "INTERNAL_ERROR",
			MessageKey: "error.common.internal",
			Err:        fmt.Errorf("share executor: %w", err),
		}
	}
}

func toAssetResultPayload(item asset.Asset) map[string]any {
	acl := decodeJSON(item.ACLJSON, []any{})
	metadata := decodeJSON(item.MetadataJSON, map[string]any{})
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         acl,
		"name":        item.Name,
		"type":        item.Type,
		"mime":        item.Mime,
		"size":        item.Size,
		"uri":         item.URI,
		"hash":        item.Hash,
		"metadata":    metadata,
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toWorkflowTemplateResultPayload(item workflow.WorkflowTemplate) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"tenantId":       item.TenantID,
		"workspaceId":    item.WorkspaceID,
		"ownerId":        item.OwnerID,
		"visibility":     item.Visibility,
		"acl":            decodeJSON(item.ACLJSON, []any{}),
		"status":         item.Status,
		"name":           item.Name,
		"description":    item.Description,
		"graph":          decodeJSON(item.GraphJSON, map[string]any{}),
		"schemaInputs":   decodeJSON(item.SchemaInputsJSON, map[string]any{}),
		"schemaOutputs":  decodeJSON(item.SchemaOutputsJSON, map[string]any{}),
		"uiState":        decodeJSON(item.UIStateJSON, map[string]any{}),
		"currentVersion": item.CurrentVersion,
		"createdAt":      item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":      item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func toWorkflowRunResultPayload(item workflow.WorkflowRun) map[string]any {
	result := map[string]any{
		"id":              item.ID,
		"tenantId":        item.TenantID,
		"workspaceId":     item.WorkspaceID,
		"ownerId":         item.OwnerID,
		"traceId":         item.TraceID,
		"visibility":      item.Visibility,
		"acl":             decodeJSON(item.ACLJSON, []any{}),
		"status":          item.Status,
		"templateId":      item.TemplateID,
		"templateVersion": item.TemplateVersion,
		"attempt":         item.Attempt,
		"inputs":          decodeJSON(item.InputsJSON, map[string]any{}),
		"outputs":         decodeJSON(item.OutputsJSON, map[string]any{}),
		"startedAt":       item.StartedAt.UTC().Format(timeRFC3339Nano),
		"createdAt":       item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":       item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.CommandID != "" {
		result["commandId"] = item.CommandID
	}
	if item.RetryOfRunID != "" {
		result["retryOfRunId"] = item.RetryOfRunID
	}
	if item.ReplayFromStepKey != "" {
		result["replayFromStepKey"] = item.ReplayFromStepKey
	}
	if item.FinishedAt != nil {
		result["finishedAt"] = item.FinishedAt.UTC().Format(timeRFC3339Nano)
		result["durationMs"] = durationMillis(item.StartedAt, *item.FinishedAt)
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		result["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return result
}

func toAlgorithmRunResultPayload(item algorithm.Run) map[string]any {
	result := map[string]any{
		"id":            item.ID,
		"algorithmId":   item.AlgorithmID,
		"workflowRunId": item.WorkflowRunID,
		"status":        item.Status,
		"outputs":       decodeJSON(item.OutputsJSON, map[string]any{}),
		"assetIds":      decodeJSON(item.AssetIDsJSON, []any{}),
		"createdAt":     item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":     item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		result["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return result
}

func toPluginPackageResultPayload(item plugin.PluginPackage) map[string]any {
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"name":        item.Name,
		"version":     item.Version,
		"packageType": item.PackageType,
		"manifest":    decodeJSON(item.ManifestJSON, map[string]any{}),
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func toPluginInstallResultPayload(item plugin.PluginInstall) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"packageId":   item.PackageID,
		"scope":       item.Scope,
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.InstalledAt != nil {
		result["installedAt"] = item.InstalledAt.UTC().Format(timeRFC3339Nano)
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		result["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return result
}

func toStreamResultPayload(item stream.Stream) map[string]any {
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"path":        item.Path,
		"protocol":    item.Protocol,
		"source":      item.Source,
		"endpoints":   decodeJSON(item.EndpointsJSON, map[string]any{}),
		"state":       decodeJSON(item.StateJSON, map[string]any{}),
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func toStreamRecordingResultPayload(item stream.Recording) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"streamId":    item.StreamID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"status":      item.Status,
		"startedAt":   item.StartedAt.UTC().Format(timeRFC3339Nano),
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.AssetID != "" {
		result["assetId"] = item.AssetID
	}
	if item.FinishedAt != nil {
		result["finishedAt"] = item.FinishedAt.UTC().Format(timeRFC3339Nano)
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		result["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return result
}

func toShareResultPayload(item command.Share) map[string]any {
	resp := map[string]any{
		"id":           item.ID,
		"tenantId":     item.TenantID,
		"workspaceId":  item.WorkspaceID,
		"resourceType": item.ResourceType,
		"resourceId":   item.ResourceID,
		"subjectType":  item.SubjectType,
		"subjectId":    item.SubjectID,
		"permissions":  item.Permissions,
		"createdBy":    item.CreatedBy,
		"createdAt":    item.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.ExpiresAt != nil {
		resp["expiresAt"] = item.ExpiresAt.UTC().Format(timeRFC3339Nano)
	}
	return resp
}

func toShareDeleteResultPayload(shareID string) map[string]any {
	return map[string]any{
		"id":     shareID,
		"status": "deleted",
	}
}

func toAISessionResultPayload(item ai.Session) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"title":       item.Title,
		"goal":        item.Goal,
		"status":      item.Status,
		"inputs":      decodeJSON(item.InputsJSON, map[string]any{}),
		"constraints": decodeJSON(item.ConstraintsJSON, map[string]any{}),
		"preferences": decodeJSON(item.PreferencesJSON, map[string]any{}),
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.ArchivedAt != nil {
		result["archivedAt"] = item.ArchivedAt.UTC().Format(timeRFC3339Nano)
	}
	if item.LastTurnAt != nil {
		result["lastTurnAt"] = item.LastTurnAt.UTC().Format(timeRFC3339Nano)
	}
	return result
}

func toAISessionTurnResultPayload(item ai.SessionTurn) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"status":      command.StatusSucceeded,
		"sessionId":   item.SessionID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"role":        item.Role,
		"content":     item.Content,
		"commandIds":  decodeJSON(item.CommandIDsJSON, []any{}),
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
	if strings.TrimSpace(item.CommandType) != "" {
		result["commandType"] = strings.TrimSpace(item.CommandType)
	}
	return result
}

func objectOrDefault(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func decodeJSON[T any](raw json.RawMessage, fallback T) T {
	if len(raw) == 0 {
		return fallback
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return fallback
	}
	return out
}

func durationMillis(startedAt time.Time, finishedAt time.Time) int64 {
	ms := finishedAt.UTC().Sub(startedAt.UTC()).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}
