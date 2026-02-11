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
import java.util.Map;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.core.AuthenticationException;
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
        return ResponseEntity.badRequest().body(ErrorEnvelope.of(
                "INVALID_REQUEST",
                "error.request.invalid",
                Map.of("reason", safeReason(ex.getMessage()))
        ));
    }

    /**
     * Maps authorization and state failures to FORBIDDEN contract error.
     */
    @ExceptionHandler(IllegalStateException.class)
    public ResponseEntity<ErrorEnvelope> handleForbidden(IllegalStateException ex) {
        return ResponseEntity.status(HttpStatus.FORBIDDEN).body(ErrorEnvelope.of(
                "FORBIDDEN",
                "error.authz.forbidden",
                Map.of("reason", safeReason(ex.getMessage()))
        ));
    }

    /**
     * Maps Spring Security access denials to FORBIDDEN contract error.
     */
    @ExceptionHandler(AccessDeniedException.class)
    public ResponseEntity<ErrorEnvelope> handleAccessDenied(AccessDeniedException ex) {
        return ResponseEntity.status(HttpStatus.FORBIDDEN).body(ErrorEnvelope.of(
                "FORBIDDEN",
                "error.authz.forbidden",
                Map.of("reason", safeReason(ex.getMessage()))
        ));
    }

    /**
     * Maps authentication failures to UNAUTHORIZED contract error.
     */
    @ExceptionHandler(AuthenticationException.class)
    public ResponseEntity<ErrorEnvelope> handleUnauthorized(AuthenticationException ex) {
        return ResponseEntity.status(HttpStatus.UNAUTHORIZED).body(ErrorEnvelope.of(
                "UNAUTHORIZED",
                "error.authn.unauthorized",
                Map.of("reason", safeReason(ex.getMessage()))
        ));
    }

    /**
     * Maps unknown failures to INTERNAL_ERROR contract error.
     */
    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorEnvelope> handleUnknown(Exception ex) {
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                .body(ErrorEnvelope.of(
                        "INTERNAL_ERROR",
                        "error.internal",
                        Map.of("reason", safeReason(ex.getMessage()))
                ));
    }

    private String safeReason(String message) {
        if (message == null || message.isBlank()) {
            return "unspecified";
        }
        return message.length() > 240 ? message.substring(0, 240) : message;
    }
}
