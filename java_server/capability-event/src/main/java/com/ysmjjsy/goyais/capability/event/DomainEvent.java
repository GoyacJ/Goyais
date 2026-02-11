/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Domain event payload used by outbox and message bus.
 */
package com.ysmjjsy.goyais.capability.event;

import java.time.Instant;
import java.util.Map;

/**
 * Represents one immutable domain event payload emitted by command pipeline.
 */
public record DomainEvent(
        String type,
        String traceId,
        Instant occurredAt,
        Map<String, Object> payload
) {
}
