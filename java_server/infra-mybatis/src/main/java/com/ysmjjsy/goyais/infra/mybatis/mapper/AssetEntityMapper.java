/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Mapper for asset persistence and permission-aware queries.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.AssetEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.AssetLineageEntity;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import java.time.Instant;
import java.util.List;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

/**
 * Executes asset CRUD, readable list queries, and ACL permission checks.
 */
public interface AssetEntityMapper extends BaseMapper<AssetEntity> {

    /**
     * Returns one asset by scope without ACL filtering.
     */
    @Select("""
            SELECT *
            FROM assets
            WHERE id = #{assetId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            LIMIT 1
            """)
    AssetEntity selectByIdInScope(
            @Param("assetId") String assetId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId
    );

    /**
     * Returns one readable asset by id.
     */
    @Select("""
            <script>
            SELECT *
            FROM assets c
            WHERE c.id = #{assetId}
              AND c.status &lt;&gt; 'deleted'
              AND ${predicate}
            LIMIT 1
            </script>
            """)
    AssetEntity selectReadableById(
            @Param("assetId") String assetId,
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Returns readable assets sorted by created_at DESC and id DESC.
     */
    @Select("""
            <script>
            SELECT *
            FROM assets c
            WHERE c.status &lt;&gt; 'deleted'
              AND ${predicate}
            ORDER BY c.created_at DESC, c.id DESC
            LIMIT #{limit}
            OFFSET #{offset}
            </script>
            """)
    List<AssetEntity> selectReadableList(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext,
            @Param("limit") int limit,
            @Param("offset") int offset
    );

    /**
     * Counts readable assets for current data-permission context.
     */
    @Select("""
            <script>
            SELECT COUNT(1)
            FROM assets c
            WHERE c.status &lt;&gt; 'deleted'
              AND ${predicate}
            </script>
            """)
    long countReadable(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Updates mutable asset fields.
     */
    @Update("""
            <script>
            UPDATE assets
            <set>
              <if test='name != null'>name = #{name},</if>
              <if test='visibility != null'>visibility = #{visibility},</if>
              <if test='metadataProvided'>metadata_json = CAST(#{metadataJson} AS jsonb),</if>
              updated_at = #{updatedAt}
            </set>
            WHERE id = #{assetId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            </script>
            """)
    int updateMutable(
            @Param("assetId") String assetId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("name") String name,
            @Param("visibility") String visibility,
            @Param("metadataJson") String metadataJson,
            @Param("metadataProvided") boolean metadataProvided,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Marks one asset as deleted.
     */
    @Update("""
            UPDATE assets
            SET status = 'deleted',
                updated_at = #{updatedAt}
            WHERE id = #{assetId}
              AND tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
            """)
    int markDeleted(
            @Param("assetId") String assetId,
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("updatedAt") Instant updatedAt
    );

    /**
     * Returns true when user subject has requested permission.
     */
    @Select("""
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'asset'
                AND a.resource_id = #{assetId}
                AND a.subject_type = 'user'
                AND a.subject_id = #{subjectId}
                AND (a.expires_at IS NULL OR a.expires_at >= #{now})
                AND a.permissions @> jsonb_build_array(#{permission})
            )
            """)
    boolean hasPermissionForUser(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("assetId") String assetId,
            @Param("subjectId") String subjectId,
            @Param("permission") String permission,
            @Param("now") Instant now
    );

    /**
     * Returns true when any role subject has requested permission.
     */
    @Select("""
            <script>
            SELECT EXISTS(
              SELECT 1
              FROM acl_entries a
              WHERE a.tenant_id = #{tenantId}
                AND a.workspace_id = #{workspaceId}
                AND a.resource_type = 'asset'
                AND a.resource_id = #{assetId}
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
            @Param("assetId") String assetId,
            @Param("roles") List<String> roles,
            @Param("permission") String permission,
            @Param("now") Instant now
    );

    /**
     * Returns lineage edges for one asset in current scope.
     */
    @Select("""
            SELECT *
            FROM asset_lineage
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND (target_asset_id = #{assetId} OR source_asset_id = #{assetId})
            ORDER BY created_at DESC, id DESC
            """)
    List<AssetLineageEntity> selectLineage(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("assetId") String assetId
    );
}
