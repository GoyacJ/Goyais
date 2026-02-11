// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package command

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, in CreateInput) (CreateResult, error)
	Get(ctx context.Context, req RequestContext, id string) (Command, error)
	GetForAccess(ctx context.Context, req RequestContext, id string) (Command, error)
	List(ctx context.Context, params ListParams) (ListResult, error)
	HasCommandPermission(ctx context.Context, req RequestContext, commandID, permission string, now time.Time) (bool, error)
	GetShareResource(ctx context.Context, req RequestContext, resourceType, resourceID string) (ShareResource, error)
	HasShareResourcePermission(ctx context.Context, req RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error)
	CreateShare(ctx context.Context, in ShareCreateInput) (Share, error)
	ListShares(ctx context.Context, params ShareListParams) (ShareListResult, error)
	DeleteShare(ctx context.Context, req RequestContext, shareID string) error
	AppendCommandEvent(ctx context.Context, req RequestContext, commandID, eventType string, payload []byte) error
	AppendAuditEvent(ctx context.Context, req RequestContext, commandID, eventType, decision, reason string, payload []byte) error
	SetStatus(ctx context.Context, req RequestContext, commandID, status string, result []byte, errorCode, messageKey string, finishedAt *time.Time) (Command, error)
}
