/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: ACL item contract model.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Set;

/**
 * Describes one ACL grant entry in API resource representations.
 */
public record AclItem(
        String subjectType,
        String subjectId,
        Set<Permission> permissions,
        Instant expiresAt
) {
}
