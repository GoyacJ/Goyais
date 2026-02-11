/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Mapper for share persistence, list, delete, and permission checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.ShareEntity;
import java.time.Instant;
import java.util.List;
import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

/**
 * Executes ACL entry operations used by share APIs.
 */
public interface ShareEntityMapper extends BaseMapper<ShareEntity> {

    /**
     * Returns share rows by scope with newest-first ordering.
     */
    @Select("""
            SELECT *
            FROM acl_entries
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            ORDER BY created_at DESC, id DESC
            LIMIT #{limit}
            OFFSET #{offset}
            """)
    List<ShareEntity> selectPage(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("limit") int limit,
            @Param("offset") int offset
    );

    /**
     * Counts share rows in current scope.
     */
    @Select("""
            SELECT COUNT(1)
            FROM acl_entries
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            """)
    long countByScope(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId
    );

    /**
     * Deletes one share created by current subject.
     */
    @Delete("""
            DELETE FROM acl_entries
            WHERE id = #{shareId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND created_by = #{createdBy}
            """)
    int deleteByIdAndCreator(
            @Param("shareId") String shareId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("createdBy") String createdBy
    );

    /**
     * Returns true when user subject has requested permission on resource.
     */
    @Select("""
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = #{resourceType}
                AND a.resource_id = #{resourceId}
                AND a.subject_type = 'user'
                AND a.subject_id = #{subjectId}
                AND (a.expires_at IS NULL OR a.expires_at >= #{now})
                AND a.permissions @> jsonb_build_array(#{permission})
            )
            """)
    boolean hasPermissionForUser(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("resourceType") String resourceType,
            @Param("resourceId") String resourceId,
            @Param("subjectId") String subjectId,
            @Param("permission") String permission,
            @Param("now") Instant now
    );

    /**
     * Returns true when any role subject has requested permission on resource.
     */
    @Select("""
            <script>
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = #{resourceType}
                AND a.resource_id = #{resourceId}
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
            @Param("resourceType") String resourceType,
            @Param("resourceId") String resourceId,
            @Param("roles") List<String> roles,
            @Param("permission") String permission,
            @Param("now") Instant now
    );
}
