/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Domain event payload used by outbox and message bus.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
