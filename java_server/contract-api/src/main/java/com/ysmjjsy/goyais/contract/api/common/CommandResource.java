/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Read model for command query and create response.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Map;

/**
 * Represents command read model returned by list and detail APIs.
 */
public record CommandResource(
        ResourceBase base,
        String commandType,
        Map<String, Object> payload,
        Instant acceptedAt,
        String traceId,
        Map<String, Object> result,
        ErrorBody error
) {
}
