/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow run event contract model for run event stream APIs.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Map;

/**
 * Represents one workflow run event emitted during run execution.
 */
public record WorkflowRunEvent(
        String id,
        String runId,
        String tenantId,
        String workspaceId,
        String stepKey,
        String eventType,
        Map<String, Object> payload,
        Instant createdAt
) {
}
