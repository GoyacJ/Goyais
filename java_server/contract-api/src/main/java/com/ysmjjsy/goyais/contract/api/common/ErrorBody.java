/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unified machine-readable error body.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
