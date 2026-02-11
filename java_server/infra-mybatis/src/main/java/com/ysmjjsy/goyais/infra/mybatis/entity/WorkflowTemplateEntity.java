/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted workflow template row mapped from workflow_templates table.
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

    @TableField("description")
    public String description;

    @TableField("status")
    public String status;

    @TableField("current_version")
    public Integer currentVersion;

    @TableField("graph")
    public String graphJson;

    @TableField("schema_inputs")
    public String schemaInputsJson;

    @TableField("schema_outputs")
    public String schemaOutputsJson;

    @TableField("ui_state")
    public String uiStateJson;

    @TableField("created_at")
    public Instant createdAt;

    @TableField("updated_at")
    public Instant updatedAt;
}
