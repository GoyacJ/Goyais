/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset contract model aligned with Go API semantics.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
