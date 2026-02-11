/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted asset row mapped from assets table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures asset persistence fields used by read/write repositories.
 */
@TableName("assets")
public final class AssetEntity {
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("tenant_id")
    public String tenantId;

    @TableField("workspace_id")
    public String workspaceId;

    @TableField("owner_id")
    public String ownerId;

    @TableField("visibility")
    public String visibility;

    @TableField("acl_json")
    public String aclJson;

    @TableField("name")
    public String name;

    @TableField("type")
    public String type;

    @TableField("mime")
    public String mime;

    @TableField("size")
    public Long size;

    @TableField("uri")
    public String uri;

    @TableField("hash")
    public String hash;

    @TableField("metadata_json")
    public String metadataJson;

    @TableField("status")
    public String status;

    @TableField("created_at")
    public Instant createdAt;

    @TableField("updated_at")
    public Instant updatedAt;
}
