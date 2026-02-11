/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Maps bootstrap exceptions into the unified error envelope.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.contract.api.common.ErrorEnvelope;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

/**
 * Converts framework exceptions into fixed contract error envelope.
 */
@RestControllerAdvice
public final class GlobalExceptionHandler {

    /**
     * Maps argument validation failures to INVALID_REQUEST contract error.
     */
    @ExceptionHandler(IllegalArgumentException.class)
    public ResponseEntity<ErrorEnvelope> handleBadRequest(IllegalArgumentException ex) {
        return ResponseEntity.badRequest().body(ErrorEnvelope.of("INVALID_REQUEST", "error.request.invalid"));
    }

    /**
     * Maps authorization and state failures to FORBIDDEN contract error.
     */
    @ExceptionHandler(IllegalStateException.class)
    public ResponseEntity<ErrorEnvelope> handleForbidden(IllegalStateException ex) {
        return ResponseEntity.status(HttpStatus.FORBIDDEN).body(ErrorEnvelope.of("FORBIDDEN", "error.authz.forbidden"));
    }

    /**
     * Maps unknown failures to INTERNAL_ERROR contract error.
     */
    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorEnvelope> handleUnknown(Exception ex) {
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                .body(ErrorEnvelope.of("INTERNAL_ERROR", "error.internal"));
    }
}
