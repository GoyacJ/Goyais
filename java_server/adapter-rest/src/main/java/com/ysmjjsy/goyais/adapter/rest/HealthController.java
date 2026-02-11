/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Health endpoints aligned with Go API aliases.
 */
package com.ysmjjsy.goyais.adapter.rest;

import java.time.Instant;
import java.util.Map;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1")
public final class HealthController {

    /**
     * Returns primary health endpoint payload.
     */
    @GetMapping("/healthz")
    public Map<String, Object> healthz() {
        return payload();
    }

    /**
     * Returns alias health endpoint payload kept for Go contract parity.
     */
    @GetMapping("/system/healthz")
    public Map<String, Object> systemHealthz() {
        return payload();
    }

    private Map<String, Object> payload() {
        return Map.of(
                "status", "ok",
                "timestamp", Instant.now().toString(),
                "mode", "minimal",
                "providers", Map.of(
                        "db", "postgres",
                        "cache", "redis",
                        "objectStore", "local",
                        "eventBus", "memory"
                )
        );
    }
}
