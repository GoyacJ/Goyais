/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Step run contract model for workflow step APIs.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Map;

/**
 * Represents one step run resource returned by workflow step APIs.
 */
public record StepRun(
        ResourceBase base,
        String runId,
        String stepKey,
        String stepType,
        int attempt,
        String traceId,
        Map<String, Object> input,
        Map<String, Object> output,
        Map<String, Object> artifacts,
        String logRef,
        Instant startedAt,
        Instant finishedAt,
        Long durationMs,
        ErrorBody error
) {
}
