/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Share contract model for ACL sharing APIs.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
