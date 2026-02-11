/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Audit event model for command and authorization traceability.
 */
package com.ysmjjsy.goyais.domain.audit;

import java.time.Instant;
import java.util.Map;

/**
 * Stores auditable authorization and execution outcomes for one command action.
 */
public record AuditEvent(
        String type,
        String traceId,
        Instant occurredAt,
        Map<String, Object> payload
) {
}
