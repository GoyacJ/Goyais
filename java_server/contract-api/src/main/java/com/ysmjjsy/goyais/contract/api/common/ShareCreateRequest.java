/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Share create request contract for ACL grant creation.
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
