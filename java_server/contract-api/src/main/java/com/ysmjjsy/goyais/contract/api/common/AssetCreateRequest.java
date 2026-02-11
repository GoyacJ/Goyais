/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset create request contract used by multipart domain sugar endpoint.
 */
package com.ysmjjsy.goyais.contract.api.common;

/**
 * Carries non-file metadata submitted to the asset create API.
 */
public record AssetCreateRequest(
        String name,
        String type,
        String mime,
        Visibility visibility
) {
}
