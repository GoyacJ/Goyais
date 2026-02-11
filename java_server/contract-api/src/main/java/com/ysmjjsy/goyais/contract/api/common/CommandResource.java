/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Read model for command query and create response.
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
