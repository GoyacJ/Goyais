/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Generic write response wrapper for resource and command ref.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

/**
 * Wraps domain resource and command reference for command-first write endpoints.
 */
public record WriteResponse<T>(
        T resource,
        CommandRef commandRef
) {
}
