/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Step run contract model for workflow step APIs.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
