/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Share contract model for ACL sharing APIs.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Set;

/**
 * Represents one persisted share grant entry.
 */
public record Share(
        String id,
        String tenantId,
        String workspaceId,
        String resourceType,
        String resourceId,
        String subjectType,
        String subjectId,
        Set<Permission> permissions,
        Instant expiresAt,
        String createdBy,
        Instant createdAt
) {
}
