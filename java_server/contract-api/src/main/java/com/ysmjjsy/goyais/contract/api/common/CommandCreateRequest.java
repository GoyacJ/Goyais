/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Canonical command create request payload.
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
