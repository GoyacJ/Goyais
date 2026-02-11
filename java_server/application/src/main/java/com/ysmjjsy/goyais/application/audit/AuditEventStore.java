/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Repository SPI for auditable command and authorization events.
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
