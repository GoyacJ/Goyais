/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persisted step run row mapped from step_runs table.
 */
package com.ysmjjsy.goyais.infra.mybatis.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.time.Instant;

/**
 * Captures workflow step run persistence fields.
 */
@TableName("step_runs")
public final class StepRunEntity {
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    @TableField("run_id")
    public String runId;

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

    @TableField("step_key")
    public String stepKey;

    @TableField("step_type")
    public String stepType;

    @TableField("attempt")
    public Integer attempt;

    @TableField("input")
    public String inputJson;

    @TableField("output")
    public String outputJson;

    @TableField("artifacts")
    public String artifactsJson;

    @TableField("log_ref")
    public String logRef;

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
