/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Command reference returned by domain sugar write APIs.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;

/**
 * References accepted command metadata returned by write APIs.
 */
public record CommandRef(
        String commandId,
        String status,
        Instant acceptedAt
) {
}
