// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package algorithm

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
	"goyais/internal/registry"
	"goyais/internal/workflow"
)

type Service struct {
	repo            Repository
	registryService *registry.Service
	workflowService *workflow.Service
	assetService    *asset.Service
}

func NewService(
	repo Repository,
	registryService *registry.Service,
	workflowService *workflow.Service,
	assetService *asset.Service,
) *Service {
	return &Service{
		repo:            repo,
		registryService: registryService,
		workflowService: workflowService,
		assetService:    assetService,
	}
}

func (s *Service) Run(ctx context.Context, in RunInput) (Run, error) {
	if s.repo == nil || s.registryService == nil || s.workflowService == nil {
		return Run{}, ErrNotImplemented
	}

	algorithmID := strings.TrimSpace(in.AlgorithmID)
	if algorithmID == "" {
		return Run{}, ErrInvalidRequest
	}

	inputs := in.Inputs
	if len(inputs) == 0 {
		inputs = json.RawMessage(`{}`)
	}
	inputMap, err := decodeJSONObject(inputs)
	if err != nil {
		return Run{}, ErrInvalidRequest
	}

	algorithm, err := s.registryService.GetAlgorithm(ctx, in.Context, algorithmID)
	if err != nil {
		return Run{}, mapRegistryError(err)
	}

	templateID := strings.TrimSpace(algorithm.TemplateRef)
	if templateID == "" {
		return Run{}, ErrInvalidRequest
	}

	defaults, err := decodeJSONObjectOrDefault(algorithm.DefaultsJSON)
	if err != nil {
		return Run{}, ErrInvalidRequest
	}
	mergedInputs := mergeObjects(defaults, inputMap)
	workflowInputs, err := json.Marshal(mergedInputs)
	if err != nil {
		return Run{}, fmt.Errorf("marshal workflow inputs: %w", err)
	}

	runVisibility := strings.TrimSpace(in.Visibility)
	if runVisibility == "" {
		runVisibility = strings.TrimSpace(algorithm.Visibility)
	}

	workflowRun, err := s.workflowService.CreateRun(
		ctx,
		in.Context,
		templateID,
		workflowInputs,
		runVisibility,
		in.Mode,
		"",
		false,
	)
	if err != nil {
		return Run{}, mapWorkflowError(err)
	}

	outputs := normalizeOutputs(workflowRun.OutputsJSON)
	assetIDs := make([]string, 0, 1)
	if s.assetService != nil {
		assetID, err := s.createResultAsset(ctx, in.Context, algorithm.ID, workflowRun, outputs)
		if err != nil {
			return Run{}, mapAssetError(err)
		}
		if strings.TrimSpace(assetID) != "" {
			assetIDs = append(assetIDs, assetID)
		}
	}

	created, err := s.repo.CreateRun(ctx, CreateRunInput{
		Context:       in.Context,
		AlgorithmID:   algorithm.ID,
		WorkflowRunID: workflowRun.ID,
		CommandID:     "",
		Visibility:    workflowRun.Visibility,
		Outputs:       outputs,
		AssetIDs:      assetIDs,
		Status:        workflowRun.Status,
		ErrorCode:     workflowRun.ErrorCode,
		MessageKey:    workflowRun.MessageKey,
		Now:           time.Now().UTC(),
	})
	if err != nil {
		return Run{}, err
	}
	return created, nil
}

func (s *Service) createResultAsset(
	ctx context.Context,
	req command.RequestContext,
	algorithmID string,
	workflowRun workflow.WorkflowRun,
	outputs json.RawMessage,
) (string, error) {
	if s.assetService == nil {
		return "", nil
	}

	payload := map[string]any{
		"algorithmId":   algorithmID,
		"workflowRunId": workflowRun.ID,
		"status":        workflowRun.Status,
		"outputs":       decodeJSONForPayload(outputs, map[string]any{}),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal algorithm result asset: %w", err)
	}

	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])

	metadata, _ := json.Marshal(map[string]any{
		"algorithmId":   algorithmID,
		"workflowRunId": workflowRun.ID,
		"type":          "algorithm-result",
	})

	created, err := s.assetService.Create(ctx, asset.CreateInput{
		Context:    req,
		Name:       sanitizeFileName(algorithmID) + "-" + time.Now().UTC().Format("20060102150405") + ".json",
		Type:       "algorithm-result",
		Mime:       "application/json",
		Size:       int64(len(raw)),
		Hash:       hash,
		Visibility: workflowRun.Visibility,
		Metadata:   metadata,
		Now:        time.Now().UTC(),
	}, raw)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func sanitizeFileName(raw string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	value := strings.TrimSpace(replacer.Replace(raw))
	if value == "" {
		return "algorithm"
	}
	return value
}

func normalizeOutputs(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	if _, err := decodeJSONObject(raw); err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty json")
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func decodeJSONObjectOrDefault(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	return decodeJSONObject(raw)
}

func mergeObjects(base map[string]any, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func decodeJSONForPayload(raw json.RawMessage, fallback any) any {
	if len(raw) == 0 {
		return fallback
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fallback
	}
	return value
}

func mapRegistryError(err error) error {
	switch {
	case errors.Is(err, registry.ErrAlgorithmNotFound):
		return ErrAlgorithmNotFound
	case errors.Is(err, registry.ErrInvalidRequest):
		return ErrInvalidRequest
	case errors.Is(err, registry.ErrNotImplemented):
		return ErrNotImplemented
	case errors.Is(err, registry.ErrForbidden):
		reason := ""
		var forbidden *registry.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &ForbiddenError{Reason: reason}
	default:
		return fmt.Errorf("registry error: %w", err)
	}
}

func mapWorkflowError(err error) error {
	switch {
	case errors.Is(err, workflow.ErrTemplateNotFound), errors.Is(err, workflow.ErrInvalidRequest):
		return ErrInvalidRequest
	case errors.Is(err, workflow.ErrNotImplemented):
		return ErrNotImplemented
	case errors.Is(err, workflow.ErrForbidden):
		reason := ""
		var forbidden *workflow.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &ForbiddenError{Reason: reason}
	default:
		return fmt.Errorf("workflow error: %w", err)
	}
}

func mapAssetError(err error) error {
	switch {
	case errors.Is(err, asset.ErrInvalidRequest):
		return ErrInvalidRequest
	case errors.Is(err, asset.ErrNotImplemented):
		return ErrNotImplemented
	case errors.Is(err, asset.ErrForbidden):
		reason := ""
		var forbidden *asset.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = forbidden.Reason
		}
		return &ForbiddenError{Reason: reason}
	default:
		return fmt.Errorf("asset error: %w", err)
	}
}
