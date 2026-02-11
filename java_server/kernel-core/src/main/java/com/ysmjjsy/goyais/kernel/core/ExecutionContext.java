/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Agent-as-user execution context propagated across services.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.core;

import java.util.Set;

/**
 * Carries agent-as-user identity and policy metadata across command execution.
 */
public record ExecutionContext(
        String tenantId,
        String workspaceId,
        String userId,
        Set<String> roles,
        String policyVersion,
        String traceId
) {
}
