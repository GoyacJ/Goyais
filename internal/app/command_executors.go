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

	"goyais/internal/algorithm"
	"goyais/internal/asset"
	"goyais/internal/command"
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

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func registerCommandExecutors(
	commandService *command.Service,
	assetService *asset.Service,
	workflowService *workflow.Service,
	pluginService *plugin.Service,
	streamService *stream.Service,
	algorithmService *algorithm.Service,
) {
	if commandService == nil {
		return
	}
	commandService.SetExecutor("share.create", newShareCreateExecutor(commandService))
	commandService.SetExecutor("share.delete", newShareDeleteExecutor(commandService))
	if assetService != nil {
		commandService.SetExecutor("asset.upload", newAssetUploadExecutor(assetService))
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
	}
	if streamService != nil {
		commandService.SetExecutor("stream.create", newStreamCreateExecutor(streamService))
		commandService.SetExecutor("stream.record.start", newStreamRecordStartExecutor(commandService, streamService))
		commandService.SetExecutor("stream.record.stop", newStreamRecordStopExecutor(streamService))
		commandService.SetExecutor("stream.kick", newStreamKickExecutor(streamService))
	}
	if algorithmService != nil {
		commandService.SetExecutor("algorithm.run", newAlgorithmRunExecutor(algorithmService))
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

func newStreamRecordStopExecutor(streamService *stream.Service) command.ExecuteFunc {
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
