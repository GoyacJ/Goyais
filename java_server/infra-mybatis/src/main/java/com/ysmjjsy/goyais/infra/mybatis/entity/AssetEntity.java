/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted asset row mapped from assets table.</p>
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
 * Captures asset persistence fields used by read/write repositories.
 */
@TableName("assets")
public final class AssetEntity {
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
    @TableField("owner_id")
    public String ownerId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("visibility")
    public String visibility;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("acl_json")
    public String aclJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("name")
    public String name;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("type")
    public String type;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("mime")
    public String mime;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("size")
    public Long size;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("uri")
    public String uri;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("hash")
    public String hash;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("metadata_json")
    public String metadataJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("status")
    public String status;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("created_at")
    public Instant createdAt;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("updated_at")
    public Instant updatedAt;
}
