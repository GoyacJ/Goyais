/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted share row mapped from acl_entries table.
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
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("tenant_id")
    public String tenantId;

    @TableField("workspace_id")
    public String workspaceId;

    @TableField("resource_type")
    public String resourceType;

    @TableField("resource_id")
    public String resourceId;

    @TableField("subject_type")
    public String subjectType;

    @TableField("subject_id")
    public String subjectId;

    @TableField("permissions")
    public String permissionsJson;

    @TableField("expires_at")
    public Instant expiresAt;

    @TableField("created_by")
    public String createdBy;

    @TableField("created_at")
    public Instant createdAt;
}
