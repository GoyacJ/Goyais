/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted audit event row mapped from audit_events table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures auditable authorization and command execution trace fields.
 */
@TableName("audit_events")
public class AuditEventEntity {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;

    @TableField("tenant_id")
    private String tenantId;

    @TableField("workspace_id")
    private String workspaceId;

    @TableField("user_id")
    private String userId;

    @TableField("trace_id")
    private String traceId;

    @TableField("event_type")
    private String eventType;

    @TableField("command_type")
    private String commandType;

    @TableField("decision")
    private String decision;

    @TableField("reason")
    private String reason;

    @TableField("payload_json")
    private String payloadJson;

    @TableField("occurred_at")
    private Instant occurredAt;

    /**
     * Returns primary key.
     */
    public Long getId() {
        return id;
    }

    /**
     * Sets primary key.
     */
    public void setId(Long id) {
        this.id = id;
    }

    /**
     * Returns tenant identifier.
     */
    public String getTenantId() {
        return tenantId;
    }

    /**
     * Sets tenant identifier.
     */
    public void setTenantId(String tenantId) {
        this.tenantId = tenantId;
    }

    /**
     * Returns workspace identifier.
     */
    public String getWorkspaceId() {
        return workspaceId;
    }

    /**
     * Sets workspace identifier.
     */
    public void setWorkspaceId(String workspaceId) {
        this.workspaceId = workspaceId;
    }

    /**
     * Returns user identifier.
     */
    public String getUserId() {
        return userId;
    }

    /**
     * Sets user identifier.
     */
    public void setUserId(String userId) {
        this.userId = userId;
    }

    /**
     * Returns trace identifier.
     */
    public String getTraceId() {
        return traceId;
    }

    /**
     * Sets trace identifier.
     */
    public void setTraceId(String traceId) {
        this.traceId = traceId;
    }

    /**
     * Returns event type.
     */
    public String getEventType() {
        return eventType;
    }

    /**
     * Sets event type.
     */
    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    /**
     * Returns command type.
     */
    public String getCommandType() {
        return commandType;
    }

    /**
     * Sets command type.
     */
    public void setCommandType(String commandType) {
        this.commandType = commandType;
    }

    /**
     * Returns allow/deny decision string.
     */
    public String getDecision() {
        return decision;
    }

    /**
     * Sets allow/deny decision string.
     */
    public void setDecision(String decision) {
        this.decision = decision;
    }

    /**
     * Returns reason string.
     */
    public String getReason() {
        return reason;
    }

    /**
     * Sets reason string.
     */
    public void setReason(String reason) {
        this.reason = reason;
    }

    /**
     * Returns JSON payload.
     */
    public String getPayloadJson() {
        return payloadJson;
    }

    /**
     * Sets JSON payload.
     */
    public void setPayloadJson(String payloadJson) {
        this.payloadJson = payloadJson;
    }

    /**
     * Returns occurrence timestamp.
     */
    public Instant getOccurredAt() {
        return occurredAt;
    }

    /**
     * Sets occurrence timestamp.
     */
    public void setOccurredAt(Instant occurredAt) {
        this.occurredAt = occurredAt;
    }
}
