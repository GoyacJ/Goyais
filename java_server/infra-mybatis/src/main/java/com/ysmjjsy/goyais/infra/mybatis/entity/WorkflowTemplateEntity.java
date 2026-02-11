/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted workflow template row mapped from workflow_templates table.</p>
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
 * Captures workflow template persistence fields.
 */
@TableName("workflow_templates")
public final class WorkflowTemplateEntity {
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
    @TableField("description")
    public String description;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("status")
    public String status;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("current_version")
    public Integer currentVersion;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("graph")
    public String graphJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("schema_inputs")
    public String schemaInputsJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("schema_outputs")
    public String schemaOutputsJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("ui_state")
    public String uiStateJson;

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
