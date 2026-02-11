/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Minimal command aggregate model for command-first flow bootstrap.
 */
package com.ysmjjsy.goyais.domain.command;

import java.time.Instant;
import java.util.Map;

/**
 * Captures persisted command aggregate state for command-first execution tracing.
 */
public record CommandAggregate(
        String id,
        String tenantId,
        String workspaceId,
        String ownerId,
        CommandStatus status,
        String commandType,
        Map<String, Object> payload,
        Instant acceptedAt,
        Instant updatedAt,
        String traceId
) {
}
