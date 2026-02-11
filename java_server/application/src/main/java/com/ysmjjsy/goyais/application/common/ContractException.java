/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Contract-oriented runtime exception carrying status and error envelope fields.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
     * @param statusCode TODO
     * @param code TODO
     * @param messageKey TODO
     * @param details TODO
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
     * @param statusCode TODO
     * @param code TODO
     * @param messageKey TODO
     * @return TODO
     */
    public static ContractException of(int statusCode, String code, String messageKey) {
        return new ContractException(statusCode, code, messageKey, Map.of());
    }

    /**
     * Creates one contract exception with details payload.
     * @param statusCode TODO
     * @param code TODO
     * @param messageKey TODO
     * @param details TODO
     * @return TODO
     */
    public static ContractException of(int statusCode, String code, String messageKey, Map<String, Object> details) {
        return new ContractException(statusCode, code, messageKey, details);
    }

    /**
     * Returns HTTP status code.
     * @return TODO
     */
    public int statusCode() {
        return statusCode;
    }

    /**
     * Returns contract error code.
     * @return TODO
     */
    public String code() {
        return code;
    }

    /**
     * Returns i18n message key.
     * @return TODO
     */
    public String messageKey() {
        return messageKey;
    }

    /**
     * Returns immutable error details payload.
     * @return TODO
     */
    public Map<String, Object> details() {
        return details;
    }
}
