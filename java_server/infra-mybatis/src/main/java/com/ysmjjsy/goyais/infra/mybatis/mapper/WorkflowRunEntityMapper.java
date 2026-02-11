/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for workflow run persistence and permission-aware queries.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowRunEntity;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import java.time.Instant;
import java.util.List;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

/**
 * Executes workflow run CRUD, read filtering, and ACL permission checks.
 */
public interface WorkflowRunEntityMapper extends BaseMapper<WorkflowRunEntity> {

    /**
     * Returns one workflow run by scope without ACL read filtering.
     */
    @Select("""
            SELECT *
            FROM workflow_runs
            WHERE id = #{runId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            LIMIT 1
            """)
    WorkflowRunEntity selectByIdInScope(
            @Param("runId") String runId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId
    );

    /**
     * Returns one readable workflow run by id.
     */
    @Select("""
            <script>
            SELECT *
            FROM workflow_runs c
            WHERE c.id = #{runId}
              AND ${predicate}
            LIMIT 1
            </script>
            """)
    WorkflowRunEntity selectReadableById(
            @Param("runId") String runId,
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Returns readable workflow runs with stable descending order.
     */
    @Select("""
            <script>
            SELECT *
            FROM workflow_runs c
            WHERE ${predicate}
            ORDER BY c.created_at DESC, c.id DESC
            LIMIT #{limit}
            OFFSET #{offset}
            </script>
            """)
    List<WorkflowRunEntity> selectReadableList(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext,
            @Param("limit") int limit,
            @Param("offset") int offset
    );

    /**
     * Counts readable workflow runs for current context.
     */
    @Select("""
            <script>
            SELECT COUNT(1)
            FROM workflow_runs c
            WHERE ${predicate}
            </script>
            """)
    long countReadable(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Cancels run when status is pending or running.
     */
    @Update("""
            UPDATE workflow_runs
            SET status = 'canceled',
                finished_at = #{finishedAt},
                updated_at = #{updatedAt},
                error_code = NULL,
                message_key = NULL
            WHERE id = #{runId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND status IN ('pending', 'running')
            """)
    int cancelActiveRun(
            @Param("runId") String runId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("finishedAt") Instant finishedAt,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Returns true when user subject has requested run permission.
     */
    @Select("""
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'workflow_run'
                AND a.resource_id = #{runId}
                AND a.subject_type = 'user'
                AND a.subject_id = #{subjectId}
                AND (a.expires_at IS NULL OR a.expires_at >= #{now})
                AND a.permissions @> jsonb_build_array(#{permission})
            )
            """)
    boolean hasPermissionForUser(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId,
            @Param("subjectId") String subjectId,
            @Param("permission") String permission,
            @Param("now") Instant now
    );

    /**
     * Returns true when any role subject has requested run permission.
     */
    @Select("""
            <script>
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'workflow_run'
                AND a.resource_id = #{runId}
                AND a.subject_type = 'role'
                AND a.subject_id IN
                <foreach collection='roles' item='role' open='(' separator=',' close=')'>
                  #{role}
                </foreach>
                AND (a.expires_at IS NULL OR a.expires_at >= #{now})
                AND a.permissions @> jsonb_build_array(#{permission})
            )
            </script>
            """)
    boolean hasPermissionForRoles(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("runId") String runId,
            @Param("roles") List<String> roles,
            @Param("permission") String permission,
            @Param("now") Instant now
    );
}
