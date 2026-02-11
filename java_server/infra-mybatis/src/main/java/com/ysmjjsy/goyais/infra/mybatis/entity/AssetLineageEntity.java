/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted asset lineage row mapped from asset_lineage table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures lineage edge fields used by asset lineage API.
 */
@TableName("asset_lineage")
public final class AssetLineageEntity {
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("tenant_id")
    public String tenantId;

    @TableField("workspace_id")
    public String workspaceId;

    @TableField("source_asset_id")
    public String sourceAssetId;

    @TableField("target_asset_id")
    public String targetAssetId;

    @TableField("run_id")
    public String runId;

    @TableField("step_id")
    public String stepId;

    @TableField("relation")
    public String relation;

    @TableField("created_at")
    public Instant createdAt;
}
