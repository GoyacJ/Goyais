/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Evaluation result indicating whether request policyVersion is stale.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
