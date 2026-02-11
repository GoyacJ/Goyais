/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset create request contract used by multipart domain sugar endpoint.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
