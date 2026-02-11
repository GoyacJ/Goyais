/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted workflow run row mapped from workflow_runs table.</p>
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
 * Captures workflow run persistence fields.
 */
@TableName("workflow_runs")
public final class WorkflowRunEntity {
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
    @TableField("acl_json")
    public String aclJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("template_id")
    public String templateId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("template_version")
    public Integer templateVersion;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("attempt")
    public Integer attempt;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("retry_of_run_id")
    public String retryOfRunId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("replay_from_step_key")
    public String replayFromStepKey;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("command_id")
    public String commandId;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("inputs")
    public String inputsJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("outputs")
    public String outputsJson;

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
