/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Authorization gate SPI for RBAC/ACL/visibility checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;

/**
 * Evaluates authorization decision for one command request.
 */
public interface AuthorizationGate {

    /**
     * Runs the authorization gate chain and returns allow/deny result.
     */
    AuthorizationDecision authorize(CommandCreateRequest request, ExecutionContext context);
}
