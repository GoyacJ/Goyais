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

	"goyais/internal/asset"
	"goyais/internal/command"
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
	TemplateID string          `json:"templateId"`
	Inputs     json.RawMessage `json:"inputs"`
	Visibility string          `json:"visibility"`
	Mode       string          `json:"mode"`
}

type workflowCancelCommandPayload struct {
	RunID string `json:"runId"`
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func registerCommandExecutors(
	commandService *command.Service,
	assetService *asset.Service,
	workflowService *workflow.Service,
) {
	if commandService == nil {
		return
	}
	if assetService != nil {
		commandService.SetExecutor("asset.upload", newAssetUploadExecutor(assetService))
	}
	if workflowService != nil {
		commandService.SetExecutor("workflow.createDraft", newWorkflowCreateDraftExecutor(workflowService))
		commandService.SetExecutor("workflow.patch", newWorkflowPatchExecutor(workflowService))
		commandService.SetExecutor("workflow.publish", newWorkflowPublishExecutor(workflowService))
		commandService.SetExecutor("workflow.run", newWorkflowRunExecutor(workflowService))
		commandService.SetExecutor("workflow.cancel", newWorkflowCancelExecutor(workflowService))
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
		"visibility":      item.Visibility,
		"acl":             decodeJSON(item.ACLJSON, []any{}),
		"status":          item.Status,
		"templateId":      item.TemplateID,
		"templateVersion": item.TemplateVersion,
		"inputs":          decodeJSON(item.InputsJSON, map[string]any{}),
		"outputs":         decodeJSON(item.OutputsJSON, map[string]any{}),
		"startedAt":       item.StartedAt.UTC().Format(timeRFC3339Nano),
		"createdAt":       item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":       item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.CommandID != "" {
		result["commandId"] = item.CommandID
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
