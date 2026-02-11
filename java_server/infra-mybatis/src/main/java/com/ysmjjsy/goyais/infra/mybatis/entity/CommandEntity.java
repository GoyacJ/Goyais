/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted command row mapped from commands table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures canonical command persistence fields for command-first query semantics.
 */
@TableName("commands")
public class CommandEntity {
    @TableId(value = "id", type = IdType.INPUT)
    private String id;

    @TableField("tenant_id")
    private String tenantId;

    @TableField("workspace_id")
    private String workspaceId;

    @TableField("owner_id")
    private String ownerId;

    @TableField("visibility")
    private String visibility;

    @TableField("status")
    private String status;

    @TableField("command_type")
    private String commandType;

    @TableField("payload_json")
    private String payloadJson;

    @TableField("trace_id")
    private String traceId;

    @TableField("result_json")
    private String resultJson;

    @TableField("error_code")
    private String errorCode;

    @TableField("error_message_key")
    private String errorMessageKey;

    @TableField("accepted_at")
    private Instant acceptedAt;

    @TableField("created_at")
    private Instant createdAt;

    @TableField("updated_at")
    private Instant updatedAt;

    /**
     * Returns command identifier.
     */
    public String getId() {
        return id;
    }

    /**
     * Sets command identifier.
     */
    public void setId(String id) {
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
     * Returns owner identifier.
     */
    public String getOwnerId() {
        return ownerId;
    }

    /**
     * Sets owner identifier.
     */
    public void setOwnerId(String ownerId) {
        this.ownerId = ownerId;
    }

    /**
     * Returns visibility value.
     */
    public String getVisibility() {
        return visibility;
    }

    /**
     * Sets visibility value.
     */
    public void setVisibility(String visibility) {
        this.visibility = visibility;
    }

    /**
     * Returns command status value.
     */
    public String getStatus() {
        return status;
    }

    /**
     * Sets command status value.
     */
    public void setStatus(String status) {
        this.status = status;
    }

    /**
     * Returns command type value.
     */
    public String getCommandType() {
        return commandType;
    }

    /**
     * Sets command type value.
     */
    public void setCommandType(String commandType) {
        this.commandType = commandType;
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
     * Returns JSON result payload.
     */
    public String getResultJson() {
        return resultJson;
    }

    /**
     * Sets JSON result payload.
     */
    public void setResultJson(String resultJson) {
        this.resultJson = resultJson;
    }

    /**
     * Returns error code.
     */
    public String getErrorCode() {
        return errorCode;
    }

    /**
     * Sets error code.
     */
    public void setErrorCode(String errorCode) {
        this.errorCode = errorCode;
    }

    /**
     * Returns error i18n key.
     */
    public String getErrorMessageKey() {
        return errorMessageKey;
    }

    /**
     * Sets error i18n key.
     */
    public void setErrorMessageKey(String errorMessageKey) {
        this.errorMessageKey = errorMessageKey;
    }

    /**
     * Returns accepted timestamp.
     */
    public Instant getAcceptedAt() {
        return acceptedAt;
    }

    /**
     * Sets accepted timestamp.
     */
    public void setAcceptedAt(Instant acceptedAt) {
        this.acceptedAt = acceptedAt;
    }

    /**
     * Returns created timestamp.
     */
    public Instant getCreatedAt() {
        return createdAt;
    }

    /**
     * Sets created timestamp.
     */
    public void setCreatedAt(Instant createdAt) {
        this.createdAt = createdAt;
    }

    /**
     * Returns updated timestamp.
     */
    public Instant getUpdatedAt() {
        return updatedAt;
    }

    /**
     * Sets updated timestamp.
     */
    public void setUpdatedAt(Instant updatedAt) {
        this.updatedAt = updatedAt;
    }
}
