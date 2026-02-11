/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Authorization decision with reason for auditable command flow.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

/**
 * Represents allow/deny result together with auditable reason.
 */
public record AuthorizationDecision(boolean allowed, String reason) {

    /**
     * Creates an allow decision with explicit reason.
     * @param reason TODO
     * @return TODO
     */
    public static AuthorizationDecision allow(String reason) {
        return new AuthorizationDecision(true, reason);
    }

    /**
     * Creates a deny decision with explicit reason.
     * @param reason TODO
     * @return TODO
     */
    public static AuthorizationDecision deny(String reason) {
        return new AuthorizationDecision(false, reason);
    }
}
