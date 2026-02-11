/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Authorization gate SPI for RBAC/ACL/visibility checks.
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
