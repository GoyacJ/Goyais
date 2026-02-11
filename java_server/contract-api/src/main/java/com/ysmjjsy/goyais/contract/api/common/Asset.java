/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset contract model aligned with Go API semantics.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Represents one asset resource returned by asset APIs.
 */
public record Asset(
        ResourceBase base,
        String name,
        String type,
        String mime,
        long size,
        String hash,
        String uri,
        Map<String, Object> metadata
) {
}
