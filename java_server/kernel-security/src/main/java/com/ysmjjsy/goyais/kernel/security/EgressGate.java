/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Egress policy gate SPI for outbound data control.
 */
package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;

/**
 * Verifies outbound data policy before side effects leave tenant boundary.
 */
public interface EgressGate {

    /**
     * Evaluates one command request against egress policy.
     */
    AuthorizationDecision verify(CommandCreateRequest request, ExecutionContext context);
}
