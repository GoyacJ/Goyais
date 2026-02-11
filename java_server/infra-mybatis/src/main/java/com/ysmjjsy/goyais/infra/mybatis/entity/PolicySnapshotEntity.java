/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted policy snapshot row mapped from policies table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures durable dynamic-authorization policy snapshot fields.
 */
@TableName("policies")
public class PolicySnapshotEntity {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;

    @TableField("tenant_id")
    private String tenantId;

    @TableField("workspace_id")
    private String workspaceId;

    @TableField("user_id")
    private String userId;

    @TableField("policy_version")
    private String policyVersion;

    @TableField("roles_json")
    private String rolesJson;

    @TableField("denied_command_types_json")
    private String deniedCommandTypesJson;

    @TableField("updated_at")
    private Instant updatedAt;

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
     * Returns policy version.
     */
    public String getPolicyVersion() {
        return policyVersion;
    }

    /**
     * Sets policy version.
     */
    public void setPolicyVersion(String policyVersion) {
        this.policyVersion = policyVersion;
    }

    /**
     * Returns serialized roles payload.
     */
    public String getRolesJson() {
        return rolesJson;
    }

    /**
     * Sets serialized roles payload.
     */
    public void setRolesJson(String rolesJson) {
        this.rolesJson = rolesJson;
    }

    /**
     * Returns serialized denied-command payload.
     */
    public String getDeniedCommandTypesJson() {
        return deniedCommandTypesJson;
    }

    /**
     * Sets serialized denied-command payload.
     */
    public void setDeniedCommandTypesJson(String deniedCommandTypesJson) {
        this.deniedCommandTypesJson = deniedCommandTypesJson;
    }

    /**
     * Returns update timestamp.
     */
    public Instant getUpdatedAt() {
        return updatedAt;
    }

    /**
     * Sets update timestamp.
     */
    public void setUpdatedAt(Instant updatedAt) {
        this.updatedAt = updatedAt;
    }
}
