/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow run contract model aligned with Go API semantics.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Map;

/**
 * Represents one workflow run resource returned by workflow run APIs.
 */
public record WorkflowRun(
        ResourceBase base,
        String templateId,
        int templateVersion,
        int attempt,
        String retryOfRunId,
        String replayFromStepKey,
        String traceId,
        Map<String, Object> inputs,
        Map<String, Object> outputs,
        Instant startedAt,
        Instant finishedAt,
        Long durationMs,
        ErrorBody error
) {
}
