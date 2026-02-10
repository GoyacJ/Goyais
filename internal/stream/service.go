package stream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/platform/eventbus"
)

type Service struct {
	repo                 Repository
	assetService         *asset.Service
	allowPrivateToPublic bool
	eventBus             eventbus.Provider
}

func NewService(repo Repository, assetService *asset.Service, allowPrivateToPublic bool) *Service {
	return &Service{
		repo:                 repo,
		assetService:         assetService,
		allowPrivateToPublic: allowPrivateToPublic,
	}
}

func (s *Service) SetEventBusProvider(provider eventbus.Provider) {
	s.eventBus = provider
}

func (s *Service) CreateStream(
	ctx context.Context,
	req command.RequestContext,
	path string,
	protocol string,
	source string,
	visibility string,
	state json.RawMessage,
) (Stream, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Stream{}, ErrInvalidRequest
	}
	normalizedProtocol, err := normalizeProtocol(protocol)
	if err != nil {
		return Stream{}, err
	}
	normalizedSource, err := normalizeSource(source)
	if err != nil {
		return Stream{}, err
	}
	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return Stream{}, err
	}
	if len(state) == 0 {
		state = json.RawMessage(`{}`)
	}
	if !isJSONObject(state) {
		return Stream{}, ErrInvalidRequest
	}

	return s.repo.CreateStream(ctx, CreateStreamInput{
		Context:    req,
		Path:       path,
		Protocol:   normalizedProtocol,
		Source:     normalizedSource,
		Visibility: normalizedVisibility,
		State:      state,
		Now:        time.Now().UTC(),
	})
}

func (s *Service) GetStream(ctx context.Context, req command.RequestContext, streamID string) (Stream, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return Stream{}, ErrInvalidRequest
	}
	item, err := s.repo.GetStreamForAccess(ctx, req, streamID)
	if err != nil {
		return Stream{}, err
	}
	allowed, reason, err := s.authorizeStream(ctx, req, item, command.PermissionRead)
	if err != nil {
		return Stream{}, err
	}
	if !allowed {
		return Stream{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) ListStreams(ctx context.Context, params StreamListParams) (StreamListResult, error) {
	return s.repo.ListStreams(ctx, params)
}

func (s *Service) StartRecording(ctx context.Context, req command.RequestContext, streamID string) (StartRecordingResult, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return StartRecordingResult{}, ErrInvalidRequest
	}
	item, err := s.repo.GetStreamForAccess(ctx, req, streamID)
	if err != nil {
		return StartRecordingResult{}, err
	}
	allowed, reason, err := s.authorizeStream(ctx, req, item, command.PermissionExecute)
	if err != nil {
		return StartRecordingResult{}, err
	}
	if !allowed {
		return StartRecordingResult{}, &ForbiddenError{Reason: reason}
	}

	if item.Status == StreamStatusRecording {
		active, err := s.repo.GetActiveRecording(ctx, req, streamID)
		if err == nil {
			return StartRecordingResult{
				Stream:               item,
				Recording:            active,
				OnPublishTemplateID:  "",
				OnPublishEventStatus: "",
				OnPublishEventError:  "",
			}, nil
		}
		if !errors.Is(err, ErrRecordingNotFound) {
			return StartRecordingResult{}, err
		}
	}

	rec, err := s.repo.CreateRecording(ctx, CreateRecordingInput{
		Context:  req,
		StreamID: streamID,
		Now:      time.Now().UTC(),
	})
	if err != nil {
		return StartRecordingResult{}, err
	}

	updatedStream, err := s.repo.UpdateStreamStatus(ctx, UpdateStreamStatusInput{
		Context:  req,
		StreamID: streamID,
		Status:   StreamStatusRecording,
		State:    item.StateJSON,
		Now:      time.Now().UTC(),
	})
	if err != nil {
		return StartRecordingResult{}, err
	}

	templateID := extractOnPublishTemplateID(updatedStream.StateJSON)
	eventStatus, eventError := s.publishOnPublishEvent(ctx, req, updatedStream, rec, templateID)
	return StartRecordingResult{
		Stream:               updatedStream,
		Recording:            rec,
		OnPublishTemplateID:  templateID,
		OnPublishEventStatus: eventStatus,
		OnPublishEventError:  eventError,
	}, nil
}

func (s *Service) StopRecording(ctx context.Context, req command.RequestContext, streamID string) (StopRecordingResult, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return StopRecordingResult{}, ErrInvalidRequest
	}
	item, err := s.repo.GetStreamForAccess(ctx, req, streamID)
	if err != nil {
		return StopRecordingResult{}, err
	}
	allowed, reason, err := s.authorizeStream(ctx, req, item, command.PermissionExecute)
	if err != nil {
		return StopRecordingResult{}, err
	}
	if !allowed {
		return StopRecordingResult{}, &ForbiddenError{Reason: reason}
	}
	if s.assetService == nil {
		return StopRecordingResult{}, ErrNotImplemented
	}

	rec, err := s.repo.GetActiveRecording(ctx, req, streamID)
	if err != nil {
		return StopRecordingResult{}, err
	}

	now := time.Now().UTC()
	recordingBytes := []byte(fmt.Sprintf("stream:%s recording:%s finished:%s", streamID, rec.ID, now.Format(time.RFC3339Nano)))
	hash := sha256.Sum256(recordingBytes)
	hashHex := hex.EncodeToString(hash[:])

	assetMetadata, _ := json.Marshal(map[string]any{
		"streamId":    streamID,
		"recordingId": rec.ID,
	})
	createdAsset, err := s.assetService.Create(ctx, asset.CreateInput{
		Context:    req,
		Name:       fmt.Sprintf("%s-%s.mp4", strings.ReplaceAll(strings.TrimPrefix(item.Path, "/"), "/", "_"), now.Format("20060102150405")),
		Type:       "stream-recording",
		Mime:       "video/mp4",
		Size:       int64(len(recordingBytes)),
		Hash:       hashHex,
		Metadata:   assetMetadata,
		Visibility: item.Visibility,
		Now:        now,
	}, recordingBytes)
	if err != nil {
		return StopRecordingResult{}, err
	}

	completed, err := s.repo.CompleteRecording(ctx, CompleteRecordingInput{
		Context:     req,
		RecordingID: rec.ID,
		AssetID:     createdAsset.ID,
		Now:         now,
	})
	if err != nil {
		return StopRecordingResult{}, err
	}

	nextState := mergeStreamState(item.StateJSON, map[string]any{
		"lastRecordingAssetId": createdAsset.ID,
		"lastRecordingId":      completed.ID,
	})
	updatedStream, err := s.repo.UpdateStreamStatus(ctx, UpdateStreamStatusInput{
		Context:  req,
		StreamID: streamID,
		Status:   StreamStatusOnline,
		State:    nextState,
		Now:      now,
	})
	if err != nil {
		return StopRecordingResult{}, err
	}

	lineageID, err := s.repo.CreateLineage(ctx, CreateLineageInput{
		Context:       req,
		TargetAssetID: createdAsset.ID,
		StepID:        completed.ID,
		Relation:      "recorded_from",
		Now:           now,
	})
	if err != nil {
		return StopRecordingResult{}, err
	}

	return StopRecordingResult{
		Stream:    updatedStream,
		Recording: completed,
		AssetID:   createdAsset.ID,
		LineageID: lineageID,
	}, nil
}

func (s *Service) KickStream(ctx context.Context, req command.RequestContext, streamID string) (Stream, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return Stream{}, ErrInvalidRequest
	}
	item, err := s.repo.GetStreamForAccess(ctx, req, streamID)
	if err != nil {
		return Stream{}, err
	}
	allowed, reason, err := s.authorizeStream(ctx, req, item, command.PermissionManage)
	if err != nil {
		return Stream{}, err
	}
	if !allowed {
		return Stream{}, &ForbiddenError{Reason: reason}
	}

	return s.repo.UpdateStreamStatus(ctx, UpdateStreamStatusInput{
		Context:  req,
		StreamID: streamID,
		Status:   StreamStatusOffline,
		State:    item.StateJSON,
		Now:      time.Now().UTC(),
	})
}

func (s *Service) authorizeStream(ctx context.Context, req command.RequestContext, item Stream, permission string) (bool, string, error) {
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
		hasPermission, err := s.repo.HasPermission(ctx, req, ResourceTypeStream, item.ID, permission, time.Now().UTC())
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

func normalizeProtocol(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "rtsp", "rtmp", "srt", "webrtc", "hls":
		return value, nil
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeSource(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "push":
		return "push", nil
	case "pull":
		return "pull", nil
	default:
		return "", ErrInvalidRequest
	}
}

func extractOnPublishTemplateID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		return ""
	}
	value, _ := state["onPublishTemplateId"].(string)
	return strings.TrimSpace(value)
}

func (s *Service) publishOnPublishEvent(
	ctx context.Context,
	req command.RequestContext,
	streamItem Stream,
	recording Recording,
	templateID string,
) (string, string) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return "", ""
	}
	if s.eventBus == nil {
		return "skipped", "event bus provider not configured"
	}
	envelope := map[string]any{
		"eventType":   "stream.on_publish",
		"eventId":     recording.ID,
		"tenantId":    req.TenantID,
		"workspaceId": req.WorkspaceID,
		"userId":      req.UserID,
		"traceId":     req.TraceID,
		"streamId":    streamItem.ID,
		"recordingId": recording.ID,
		"templateId":  templateID,
		"visibility":  streamItem.Visibility,
		"trigger":     "stream.onPublish",
		"emittedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "failed", err.Error()
	}
	err = s.eventBus.Publish(ctx, eventbus.ChannelStream, eventbus.Message{
		Key:   recording.ID,
		Value: raw,
		Headers: map[string]string{
			"eventType":   "stream.on_publish",
			"tenantId":    req.TenantID,
			"workspaceId": req.WorkspaceID,
		},
	})
	if err != nil {
		return "failed", err.Error()
	}
	return "published", ""
}

func mergeStreamState(raw json.RawMessage, patch map[string]any) json.RawMessage {
	state := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &state)
	}
	for k, v := range patch {
		state[k] = v
	}
	merged, err := json.Marshal(state)
	if err != nil {
		return raw
	}
	return merged
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}
