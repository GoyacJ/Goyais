/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted workflow run row mapped from workflow_runs table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures workflow run persistence fields.
 */
@TableName("workflow_runs")
public final class WorkflowRunEntity {
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("tenant_id")
    public String tenantId;

    @TableField("workspace_id")
    public String workspaceId;

    @TableField("owner_id")
    public String ownerId;

    @TableField("trace_id")
    public String traceId;

    @TableField("visibility")
    public String visibility;

    @TableField("acl_json")
    public String aclJson;

    @TableField("template_id")
    public String templateId;

    @TableField("template_version")
    public Integer templateVersion;

    @TableField("attempt")
    public Integer attempt;

    @TableField("retry_of_run_id")
    public String retryOfRunId;

    @TableField("replay_from_step_key")
    public String replayFromStepKey;

    @TableField("command_id")
    public String commandId;

    @TableField("inputs")
    public String inputsJson;

    @TableField("outputs")
    public String outputsJson;

    @TableField("status")
    public String status;

    @TableField("error_code")
    public String errorCode;

    @TableField("message_key")
    public String messageKey;

    @TableField("started_at")
    public Instant startedAt;

    @TableField("finished_at")
    public Instant finishedAt;

    @TableField("created_at")
    public Instant createdAt;

    @TableField("updated_at")
    public Instant updatedAt;
}
