/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for workflow template persistence and permission-aware queries.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowTemplateEntity;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import java.time.Instant;
import java.util.List;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

/**
 * Executes workflow template CRUD, read filtering, and ACL permission checks.
 */
public interface WorkflowTemplateEntityMapper extends BaseMapper<WorkflowTemplateEntity> {

    /**
     * Returns one workflow template by scope without ACL read filtering.
     */
    @Select("""
            SELECT *
            FROM workflow_templates
            WHERE id = #{templateId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            LIMIT 1
            """)
    WorkflowTemplateEntity selectByIdInScope(
            @Param("templateId") String templateId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId
    );

    /**
     * Returns one readable workflow template by id.
     */
    @Select("""
            <script>
            SELECT *
            FROM workflow_templates c
            WHERE c.id = #{templateId}
              AND ${predicate}
            LIMIT 1
            </script>
            """)
    WorkflowTemplateEntity selectReadableById(
            @Param("templateId") String templateId,
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Returns readable workflow template list with stable descending order.
     */
    @Select("""
            <script>
            SELECT *
            FROM workflow_templates c
            WHERE ${predicate}
            ORDER BY c.created_at DESC, c.id DESC
            LIMIT #{limit}
            OFFSET #{offset}
            </script>
            """)
    List<WorkflowTemplateEntity> selectReadableList(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext,
            @Param("limit") int limit,
            @Param("offset") int offset
    );

    /**
     * Counts readable workflow templates for current context.
     */
    @Select("""
            <script>
            SELECT COUNT(1)
            FROM workflow_templates c
            WHERE ${predicate}
            </script>
            """)
    long countReadable(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Updates workflow template graph and ui_state.
     */
    @Update("""
            UPDATE workflow_templates
            SET graph = CAST(#{graphJson} AS jsonb),
                ui_state = CAST(#{uiStateJson} AS jsonb),
                updated_at = #{updatedAt}
            WHERE id = #{templateId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            """)
    int patchTemplate(
            @Param("templateId") String templateId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("graphJson") String graphJson,
            @Param("uiStateJson") String uiStateJson,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Publishes workflow template and bumps current version.
     */
    @Update("""
            UPDATE workflow_templates
            SET status = 'published',
                current_version = #{nextVersion},
                updated_at = #{updatedAt}
            WHERE id = #{templateId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            """)
    int publishTemplate(
            @Param("templateId") String templateId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("nextVersion") int nextVersion,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Inserts one immutable template version snapshot row.
     */
    @Insert("""
            INSERT INTO workflow_template_versions(
                id,
                template_id,
                version,
                graph,
                schema_inputs,
                schema_outputs,
                checksum,
                created_by,
                created_at
            )
            VALUES(
                #{id},
                #{templateId},
                #{version},
                CAST(#{graphJson} AS jsonb),
                CAST(#{schemaInputsJson} AS jsonb),
                CAST(#{schemaOutputsJson} AS jsonb),
                #{checksum},
                #{createdBy},
                #{createdAt}
            )
            """)
    int insertTemplateVersion(
            @Param("id") String id,
            @Param("templateId") String templateId,
            @Param("version") int version,
            @Param("graphJson") String graphJson,
            @Param("schemaInputsJson") String schemaInputsJson,
            @Param("schemaOutputsJson") String schemaOutputsJson,
            @Param("checksum") String checksum,
            @Param("createdBy") String createdBy,
            @Param("createdAt") Instant createdAt
    );

    /**
     * Returns true when user subject has requested template permission.
     */
    @Select("""
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'workflow_template'
                AND a.resource_id = #{templateId}
                AND a.subject_type = 'user'
                AND a.subject_id = #{subjectId}
                AND (a.expires_at IS NULL OR a.expires_at >= #{now})
                AND a.permissions @> jsonb_build_array(#{permission})
            )
            """)
    boolean hasPermissionForUser(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("templateId") String templateId,
            @Param("subjectId") String subjectId,
            @Param("permission") String permission,
            @Param("now") Instant now
    );

    /**
     * Returns true when any role subject has requested template permission.
     */
    @Select("""
            <script>
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'workflow_template'
                AND a.resource_id = #{templateId}
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
            @Param("templateId") String templateId,
            @Param("roles") List<String> roles,
            @Param("permission") String permission,
            @Param("now") Instant now
    );
}
