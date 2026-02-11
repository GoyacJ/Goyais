/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Share delete result contract model.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

/**
 * Represents delete confirmation returned by share delete endpoint.
 */
public record ShareDeleteResult(
        String id,
        String status
) {
}
