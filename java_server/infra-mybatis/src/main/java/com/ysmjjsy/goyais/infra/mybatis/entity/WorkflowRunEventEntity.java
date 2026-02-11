/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted workflow run event row mapped from workflow_run_events table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures workflow run event persistence fields.
 */
@TableName("workflow_run_events")
public final class WorkflowRunEventEntity {
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("run_id")
    public String runId;

    @TableField("tenant_id")
    public String tenantId;

    @TableField("workspace_id")
    public String workspaceId;

    @TableField("step_key")
    public String stepKey;

    @TableField("event_type")
    public String eventType;

    @TableField("payload")
    public String payloadJson;

    @TableField("created_at")
    public Instant createdAt;
}
