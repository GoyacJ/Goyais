/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Policy invalidation event broadcast to all resource server nodes.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

import java.time.Instant;

/**
 * Carries the minimum payload required to invalidate dynamic policy caches.
 */
public record PolicyInvalidationEvent(
        String tenantId,
        String workspaceId,
        String userId,
        String policyVersion,
        String traceId,
        Instant changedAt
) {
}
