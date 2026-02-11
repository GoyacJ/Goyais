/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Egress policy gate SPI for outbound data control.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
