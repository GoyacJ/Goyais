/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted audit event row mapped from audit_events table.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
     * @return TODO
     */
    public Long getId() {
        return id;
    }

    /**
     * Sets primary key.
     * @param id TODO
     */
    public void setId(Long id) {
        this.id = id;
    }

    /**
     * Returns tenant identifier.
     * @return TODO
     */
    public String getTenantId() {
        return tenantId;
    }

    /**
     * Sets tenant identifier.
     * @param tenantId TODO
     */
    public void setTenantId(String tenantId) {
        this.tenantId = tenantId;
    }

    /**
     * Returns workspace identifier.
     * @return TODO
     */
    public String getWorkspaceId() {
        return workspaceId;
    }

    /**
     * Sets workspace identifier.
     * @param workspaceId TODO
     */
    public void setWorkspaceId(String workspaceId) {
        this.workspaceId = workspaceId;
    }

    /**
     * Returns user identifier.
     * @return TODO
     */
    public String getUserId() {
        return userId;
    }

    /**
     * Sets user identifier.
     * @param userId TODO
     */
    public void setUserId(String userId) {
        this.userId = userId;
    }

    /**
     * Returns trace identifier.
     * @return TODO
     */
    public String getTraceId() {
        return traceId;
    }

    /**
     * Sets trace identifier.
     * @param traceId TODO
     */
    public void setTraceId(String traceId) {
        this.traceId = traceId;
    }

    /**
     * Returns event type.
     * @return TODO
     */
    public String getEventType() {
        return eventType;
    }

    /**
     * Sets event type.
     * @param eventType TODO
     */
    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    /**
     * Returns command type.
     * @return TODO
     */
    public String getCommandType() {
        return commandType;
    }

    /**
     * Sets command type.
     * @param commandType TODO
     */
    public void setCommandType(String commandType) {
        this.commandType = commandType;
    }

    /**
     * Returns allow/deny decision string.
     * @return TODO
     */
    public String getDecision() {
        return decision;
    }

    /**
     * Sets allow/deny decision string.
     * @param decision TODO
     */
    public void setDecision(String decision) {
        this.decision = decision;
    }

    /**
     * Returns reason string.
     * @return TODO
     */
    public String getReason() {
        return reason;
    }

    /**
     * Sets reason string.
     * @param reason TODO
     */
    public void setReason(String reason) {
        this.reason = reason;
    }

    /**
     * Returns JSON payload.
     * @return TODO
     */
    public String getPayloadJson() {
        return payloadJson;
    }

    /**
     * Sets JSON payload.
     * @param payloadJson TODO
     */
    public void setPayloadJson(String payloadJson) {
        this.payloadJson = payloadJson;
    }

    /**
     * Returns occurrence timestamp.
     * @return TODO
     */
    public Instant getOccurredAt() {
        return occurredAt;
    }

    /**
     * Sets occurrence timestamp.
     * @param occurredAt TODO
     */
    public void setOccurredAt(Instant occurredAt) {
        this.occurredAt = occurredAt;
    }
}
