/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted policy snapshot row mapped from policies table.</p>
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
     * Returns policy version.
     * @return TODO
     */
    public String getPolicyVersion() {
        return policyVersion;
    }

    /**
     * Sets policy version.
     * @param policyVersion TODO
     */
    public void setPolicyVersion(String policyVersion) {
        this.policyVersion = policyVersion;
    }

    /**
     * Returns serialized roles payload.
     * @return TODO
     */
    public String getRolesJson() {
        return rolesJson;
    }

    /**
     * Sets serialized roles payload.
     * @param rolesJson TODO
     */
    public void setRolesJson(String rolesJson) {
        this.rolesJson = rolesJson;
    }

    /**
     * Returns serialized denied-command payload.
     * @return TODO
     */
    public String getDeniedCommandTypesJson() {
        return deniedCommandTypesJson;
    }

    /**
     * Sets serialized denied-command payload.
     * @param deniedCommandTypesJson TODO
     */
    public void setDeniedCommandTypesJson(String deniedCommandTypesJson) {
        this.deniedCommandTypesJson = deniedCommandTypesJson;
    }

    /**
     * Returns update timestamp.
     * @return TODO
     */
    public Instant getUpdatedAt() {
        return updatedAt;
    }

    /**
     * Sets update timestamp.
     * @param updatedAt TODO
     */
    public void setUpdatedAt(Instant updatedAt) {
        this.updatedAt = updatedAt;
    }
}
