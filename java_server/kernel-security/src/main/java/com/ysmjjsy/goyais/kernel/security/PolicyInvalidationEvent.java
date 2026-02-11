/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Policy invalidation event broadcast to all resource server nodes.
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
