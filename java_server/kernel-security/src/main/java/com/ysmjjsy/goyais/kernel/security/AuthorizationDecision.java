/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Authorization decision with reason for auditable command flow.
 */
package com.ysmjjsy.goyais.kernel.security;

/**
 * Represents allow/deny result together with auditable reason.
 */
public record AuthorizationDecision(boolean allowed, String reason) {

    /**
     * Creates an allow decision with explicit reason.
     */
    public static AuthorizationDecision allow(String reason) {
        return new AuthorizationDecision(true, reason);
    }

    /**
     * Creates a deny decision with explicit reason.
     */
    public static AuthorizationDecision deny(String reason) {
        return new AuthorizationDecision(false, reason);
    }
}
