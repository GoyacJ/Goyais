/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Evaluation result indicating whether request policyVersion is stale.
 */
package com.ysmjjsy.goyais.kernel.security;

import java.time.Instant;

/**
 * Represents policy version freshness in one authorization evaluation.
 */
public record PolicyVersionState(
        String requestVersion,
        String effectiveVersion,
        boolean stale,
        Instant evaluatedAt
) {
}
