/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted command row mapped from commands table.</p>
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
     * @return TODO
     */
    public String getId() {
        return id;
    }

    /**
     * Sets command identifier.
     * @param id TODO
     */
    public void setId(String id) {
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
     * Returns owner identifier.
     * @return TODO
     */
    public String getOwnerId() {
        return ownerId;
    }

    /**
     * Sets owner identifier.
     * @param ownerId TODO
     */
    public void setOwnerId(String ownerId) {
        this.ownerId = ownerId;
    }

    /**
     * Returns visibility value.
     * @return TODO
     */
    public String getVisibility() {
        return visibility;
    }

    /**
     * Sets visibility value.
     * @param visibility TODO
     */
    public void setVisibility(String visibility) {
        this.visibility = visibility;
    }

    /**
     * Returns command status value.
     * @return TODO
     */
    public String getStatus() {
        return status;
    }

    /**
     * Sets command status value.
     * @param status TODO
     */
    public void setStatus(String status) {
        this.status = status;
    }

    /**
     * Returns command type value.
     * @return TODO
     */
    public String getCommandType() {
        return commandType;
    }

    /**
     * Sets command type value.
     * @param commandType TODO
     */
    public void setCommandType(String commandType) {
        this.commandType = commandType;
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
     * Returns JSON result payload.
     * @return TODO
     */
    public String getResultJson() {
        return resultJson;
    }

    /**
     * Sets JSON result payload.
     * @param resultJson TODO
     */
    public void setResultJson(String resultJson) {
        this.resultJson = resultJson;
    }

    /**
     * Returns error code.
     * @return TODO
     */
    public String getErrorCode() {
        return errorCode;
    }

    /**
     * Sets error code.
     * @param errorCode TODO
     */
    public void setErrorCode(String errorCode) {
        this.errorCode = errorCode;
    }

    /**
     * Returns error i18n key.
     * @return TODO
     */
    public String getErrorMessageKey() {
        return errorMessageKey;
    }

    /**
     * Sets error i18n key.
     * @param errorMessageKey TODO
     */
    public void setErrorMessageKey(String errorMessageKey) {
        this.errorMessageKey = errorMessageKey;
    }

    /**
     * Returns accepted timestamp.
     * @return TODO
     */
    public Instant getAcceptedAt() {
        return acceptedAt;
    }

    /**
     * Sets accepted timestamp.
     * @param acceptedAt TODO
     */
    public void setAcceptedAt(Instant acceptedAt) {
        this.acceptedAt = acceptedAt;
    }

    /**
     * Returns created timestamp.
     * @return TODO
     */
    public Instant getCreatedAt() {
        return createdAt;
    }

    /**
     * Sets created timestamp.
     * @param createdAt TODO
     */
    public void setCreatedAt(Instant createdAt) {
        this.createdAt = createdAt;
    }

    /**
     * Returns updated timestamp.
     * @return TODO
     */
    public Instant getUpdatedAt() {
        return updatedAt;
    }

    /**
     * Sets updated timestamp.
     * @param updatedAt TODO
     */
    public void setUpdatedAt(Instant updatedAt) {
        this.updatedAt = updatedAt;
    }
}
