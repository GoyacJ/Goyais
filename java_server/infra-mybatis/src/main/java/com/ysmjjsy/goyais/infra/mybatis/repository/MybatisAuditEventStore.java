/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: MyBatisPlus implementation for persisting command audit events.
 */
package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.audit.AuditEventStore;
import com.ysmjjsy.goyais.domain.audit.AuditEvent;
import com.ysmjjsy.goyais.infra.mybatis.entity.AuditEventEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.AuditEventEntityMapper;
import java.util.Map;
import org.springframework.stereotype.Repository;

/**
 * Stores authorization and execution audit records in audit_events table.
 */
@Repository
public final class MybatisAuditEventStore implements AuditEventStore {
    private final AuditEventEntityMapper mapper;
    private final ObjectMapper objectMapper;

    /**
     * Creates store with mapper and JSON codec dependencies.
     */
    public MybatisAuditEventStore(AuditEventEntityMapper mapper, ObjectMapper objectMapper) {
        this.mapper = mapper;
        this.objectMapper = objectMapper;
    }

    /**
     * Persists one audit event row.
     */
    @Override
    public void save(AuditEvent event) {
        AuditEventEntity entity = new AuditEventEntity();
        entity.setTenantId(payloadString(event.payload(), "tenantId"));
        entity.setWorkspaceId(payloadString(event.payload(), "workspaceId"));
        entity.setUserId(payloadString(event.payload(), "userId"));
        entity.setTraceId(event.traceId());
        entity.setEventType(event.type());
        entity.setCommandType(payloadString(event.payload(), "commandType"));
        entity.setDecision(payloadString(event.payload(), "decision"));
        entity.setReason(payloadString(event.payload(), "reason"));
        entity.setPayloadJson(writeJson(event.payload()));
        entity.setOccurredAt(event.occurredAt());
        mapper.insert(entity);
    }

    private String payloadString(Map<String, Object> payload, String key) {
        if (payload == null) {
            return null;
        }
        Object value = payload.get(key);
        return value == null ? null : String.valueOf(value);
    }

    private String writeJson(Map<String, Object> payload) {
        if (payload == null) {
            return null;
        }
        try {
            return objectMapper.writeValueAsString(payload);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize audit payload", ex);
        }
    }
}
