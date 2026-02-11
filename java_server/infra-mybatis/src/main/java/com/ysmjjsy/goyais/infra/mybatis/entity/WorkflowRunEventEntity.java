/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Persisted workflow run event row mapped from workflow_run_events table.</p>
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
 * Captures workflow run event persistence fields.
 */
@TableName("workflow_run_events")
public final class WorkflowRunEventEntity {
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
    @TableField("step_key")
    public String stepKey;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("event_type")
    public String eventType;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("payload")
    public String payloadJson;

    /**
     * <p>TODO: describe field.</p>
     */
    @TableField("created_at")
    public Instant createdAt;
}
