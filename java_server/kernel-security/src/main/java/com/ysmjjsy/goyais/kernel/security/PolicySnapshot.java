/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Effective policy snapshot used by dynamic authorization decisions.
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
