/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>SQL-layer data permission context for row-level filtering.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.mybatis;

import java.util.Set;

/**
 * Carries caller scope metadata used to compose SQL row-level predicates.
 */
public record DataPermissionContext(
        String tenantId,
        String workspaceId,
        String userId,
        Set<String> roles,
        String policyVersion,
        String resourceType,
        String requiredPermission
) {
}
