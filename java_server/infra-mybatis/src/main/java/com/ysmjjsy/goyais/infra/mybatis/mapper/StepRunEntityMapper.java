/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for step run persistence and list queries.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.StepRunEntity;
import java.time.Instant;
import java.util.List;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

/**
 * Executes step run list and status update operations.
 */
public interface StepRunEntityMapper extends BaseMapper<StepRunEntity> {

    /**
     * Returns step runs for one run with stable descending order.
     */
    @Select("""
            SELECT *
            FROM step_runs
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND run_id = #{runId}
            ORDER BY created_at DESC, id DESC
            LIMIT #{limit}
            OFFSET #{offset}
            """)
    List<StepRunEntity> selectByRun(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId,
            @Param("limit") int limit,
            @Param("offset") int offset
    );

    /**
     * Counts step runs for one run in scope.
     */
    @Select("""
            SELECT COUNT(1)
            FROM step_runs
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND run_id = #{runId}
            """)
    long countByRun(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId
    );

    /**
     * Cancels step runs currently pending or running.
     */
    @Update("""
            UPDATE step_runs
            SET status = 'canceled',
                finished_at = #{finishedAt},
                updated_at = #{updatedAt},
                error_code = NULL,
                message_key = NULL
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND run_id = #{runId}
              AND status IN ('pending', 'running')
            """)
    int cancelActiveSteps(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId,
            @Param("finishedAt") Instant finishedAt,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Returns step keys currently pending/running in ascending creation order.
     */
    @Select("""
            SELECT step_key
            FROM step_runs
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND run_id = #{runId}
              AND status IN ('pending', 'running')
            ORDER BY created_at ASC, id ASC
            """)
    List<String> selectActiveStepKeys(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId
    );
}
