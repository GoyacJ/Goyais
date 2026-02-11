/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Contract-oriented runtime exception carrying status and error envelope fields.
 */
package com.ysmjjsy.goyais.application.common;

import java.util.Map;

/**
 * Represents one API contract failure with explicit HTTP status and error metadata.
 */
public final class ContractException extends RuntimeException {
    private final int statusCode;
    private final String code;
    private final String messageKey;
    private final Map<String, Object> details;

    /**
     * Creates one contract exception with status, code, message key, and optional details.
     */
    public ContractException(int statusCode, String code, String messageKey, Map<String, Object> details) {
        super(code + ":" + messageKey);
        this.statusCode = statusCode;
        this.code = code;
        this.messageKey = messageKey;
        this.details = details == null ? Map.of() : Map.copyOf(details);
    }

    /**
     * Creates one contract exception without details payload.
     */
    public static ContractException of(int statusCode, String code, String messageKey) {
        return new ContractException(statusCode, code, messageKey, Map.of());
    }

    /**
     * Creates one contract exception with details payload.
     */
    public static ContractException of(int statusCode, String code, String messageKey, Map<String, Object> details) {
        return new ContractException(statusCode, code, messageKey, details);
    }

    /**
     * Returns HTTP status code.
     */
    public int statusCode() {
        return statusCode;
    }

    /**
     * Returns contract error code.
     */
    public String code() {
        return code;
    }

    /**
     * Returns i18n message key.
     */
    public String messageKey() {
        return messageKey;
    }

    /**
     * Returns immutable error details payload.
     */
    public Map<String, Object> details() {
        return details;
    }
}
