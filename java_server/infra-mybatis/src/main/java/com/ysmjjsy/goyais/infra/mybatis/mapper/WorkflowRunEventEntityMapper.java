/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Mapper for workflow run event persistence and ordered read queries.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowRunEventEntity;
import java.util.List;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

/**
 * Executes workflow run event queries with deterministic ascending ordering.
 */
public interface WorkflowRunEventEntityMapper extends BaseMapper<WorkflowRunEventEntity> {

    /**
     * Returns run events for one run in scope ordered by created_at ASC and id ASC.
     */
    @Select("""
            SELECT *
            FROM workflow_run_events
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND run_id = #{runId}
            ORDER BY created_at ASC, id ASC
            """)
    List<WorkflowRunEventEntity> selectByRun(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId
    );
}
