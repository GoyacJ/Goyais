/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Shared contract fields for primary resources.
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
