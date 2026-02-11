/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Audit event model for command and authorization traceability.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
