/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Shared contract fields for primary resources.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.List;

/**
 * Defines shared resource fields required by Go and Java contract parity.
 */
public record ResourceBase(
        String id,
        String tenantId,
        String workspaceId,
        String ownerId,
        Visibility visibility,
        List<AclItem> acl,
        String status,
        Instant createdAt,
        Instant updatedAt
) {
}
