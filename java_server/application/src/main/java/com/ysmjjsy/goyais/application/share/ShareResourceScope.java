/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Minimal resource scope projection used by share authorization checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.share;

/**
 * Captures resource owner and scope fields needed before creating a share grant.
 */
public record ShareResourceScope(
        String resourceType,
        String resourceId,
        String tenantId,
        String workspaceId,
        String ownerId,
        String visibility
) {
}
