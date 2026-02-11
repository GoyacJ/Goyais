/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow run event contract model for run event stream APIs.
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
