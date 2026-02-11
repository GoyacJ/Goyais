/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Fixed error envelope compatible with Go contract.
 */
package com.ysmjjsy.goyais.contract.api.common;

/**
 * Wraps all API errors using the fixed Go-compatible envelope contract.
 */
public record ErrorEnvelope(ErrorBody error) {

    /**
     * Builds one error envelope from code and message key.
     */
    public static ErrorEnvelope of(String code, String messageKey) {
        return new ErrorEnvelope(new ErrorBody(code, messageKey, null));
    }
}
