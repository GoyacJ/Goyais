/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted step run row mapped from step_runs table.</p>
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
 * Captures workflow step run persistence fields.
 */
@TableName("step_runs")
public final class StepRunEntity {
    /**
     * <p>TODO: describe field.</p>
     */
    @TableId(value = "id", type = IdType.INPUT)
    public String id;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("run_id")
    public String runId;

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
    @TableField("trace_id")
    public String traceId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("visibility")
    public String visibility;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("step_key")
    public String stepKey;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("step_type")
    public String stepType;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("attempt")
    public Integer attempt;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("input")
    public String inputJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("output")
    public String outputJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("artifacts")
    public String artifactsJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("log_ref")
    public String logRef;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("status")
    public String status;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("error_code")
    public String errorCode;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("message_key")
    public String messageKey;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("started_at")
    public Instant startedAt;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("finished_at")
    public Instant finishedAt;

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
