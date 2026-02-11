/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Minimal resource scope projection used by share authorization checks.
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
