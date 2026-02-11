/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Share create request contract for ACL grant creation.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Set;

/**
 * Carries fields required to create one share ACL entry.
 */
public record ShareCreateRequest(
        String resourceType,
        String resourceId,
        String subjectType,
        String subjectId,
        Set<Permission> permissions,
        Instant expiresAt
) {
}
