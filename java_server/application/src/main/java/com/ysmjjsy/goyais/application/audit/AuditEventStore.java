/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Repository SPI for auditable command and authorization events.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.audit;

import com.ysmjjsy.goyais.domain.audit.AuditEvent;

/**
 * Persists audit events so command authorization and execution can be traced.
 */
public interface AuditEventStore {

    /**
     * Persists one audit event atomically.
     */
    void save(AuditEvent event);
}
