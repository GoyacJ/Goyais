/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: SQL-layer data permission context for row-level filtering.
 */
package com.ysmjjsy.goyais.kernel.mybatis;

import java.util.Set;

public record DataPermissionContext(
        String tenantId,
        String workspaceId,
        String userId,
        Set<String> roles
) {
}
