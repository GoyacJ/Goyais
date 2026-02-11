/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Fixed error envelope compatible with Go contract.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Wraps all API errors using the fixed Go-compatible envelope contract.
 */
public record ErrorEnvelope(ErrorBody error) {

    /**
     * Builds one error envelope from code and message key.
     * @param code TODO
     * @param messageKey TODO
     * @return TODO
     */
    public static ErrorEnvelope of(String code, String messageKey) {
        return new ErrorEnvelope(new ErrorBody(code, messageKey, null));
    }

    /**
     * Builds one error envelope from code, message key, and details payload.
     * @param code TODO
     * @param messageKey TODO
     * @param details TODO
     * @return TODO
     */
    public static ErrorEnvelope of(String code, String messageKey, Map<String, Object> details) {
        return new ErrorEnvelope(new ErrorBody(code, messageKey, details));
    }
}
