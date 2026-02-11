/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted asset lineage row mapped from asset_lineage table.</p>
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
 * Captures lineage edge fields used by asset lineage API.
 */
@TableName("asset_lineage")
public final class AssetLineageEntity {
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
    @TableField("source_asset_id")
    public String sourceAssetId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("target_asset_id")
    public String targetAssetId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("run_id")
    public String runId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("step_id")
    public String stepId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("relation")
    public String relation;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("created_at")
    public Instant createdAt;
}
