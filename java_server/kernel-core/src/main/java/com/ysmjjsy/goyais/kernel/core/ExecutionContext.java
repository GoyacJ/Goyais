/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Agent-as-user execution context propagated across services.
 */
package com.ysmjjsy.goyais.kernel.core;

import java.util.Set;

public record ExecutionContext(
        String tenantId,
        String workspaceId,
        String userId,
        Set<String> roles,
        String policyVersion,
        String traceId
) {
}
