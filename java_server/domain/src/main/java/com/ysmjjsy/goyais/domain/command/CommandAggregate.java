/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Minimal command aggregate model for command-first flow bootstrap.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
