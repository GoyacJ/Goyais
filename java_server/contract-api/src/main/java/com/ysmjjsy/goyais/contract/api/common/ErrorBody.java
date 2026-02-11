/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unified machine-readable error body.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Describes machine-readable error payload for i18n-aware clients.
 */
public record ErrorBody(
        String code,
        String messageKey,
        Map<String, Object> details
) {
}
