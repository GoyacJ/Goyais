/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Canonical command create request payload.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Defines canonical payload submitted to `POST /api/v1/commands`.
 */
public record CommandCreateRequest(
        String commandType,
        Map<String, Object> payload,
        Visibility visibility
) {
}
