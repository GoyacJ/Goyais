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

func registerCommandExecutors(commandService *command.Service, assetService *asset.Service) {
	if commandService == nil || assetService == nil {
		return
	}
	commandService.SetExecutor("asset.upload", newAssetUploadExecutor(assetService))
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
