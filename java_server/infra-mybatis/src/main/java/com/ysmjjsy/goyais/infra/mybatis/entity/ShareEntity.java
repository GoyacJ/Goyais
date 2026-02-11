/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted share row mapped from acl_entries table.</p>
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
 * Captures share persistence fields for ACL grants.
 */
@TableName("acl_entries")
public final class ShareEntity {
    /**
     * <p>TODO: describe field.</p>
     */
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("tenant_id")
    public String tenantId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("workspace_id")
    public String workspaceId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("resource_type")
    public String resourceType;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("resource_id")
    public String resourceId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("subject_type")
    public String subjectType;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("subject_id")
    public String subjectId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("permissions")
    public String permissionsJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("expires_at")
    public Instant expiresAt;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("created_by")
    public String createdBy;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("created_at")
    public Instant createdAt;
}
