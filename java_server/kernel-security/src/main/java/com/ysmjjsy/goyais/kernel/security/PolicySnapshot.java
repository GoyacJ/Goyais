/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Effective policy snapshot used by dynamic authorization decisions.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

import java.time.Instant;
import java.util.Set;

/**
 * Captures the effective policy materialized for one user scope.
 */
public record PolicySnapshot(
        String tenantId,
        String workspaceId,
        String userId,
        String policyVersion,
        Set<String> roles,
        Set<String> deniedCommandTypes,
        Instant updatedAt
) {
}
