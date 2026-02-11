/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for policy snapshot load and upsert operations.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.PolicySnapshotEntity;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

/**
 * Provides SQL operations for dynamic-authorization policy snapshot persistence.
 */
public interface PolicySnapshotEntityMapper extends BaseMapper<PolicySnapshotEntity> {

    /**
     * Returns one policy snapshot row by tenant/workspace/user scope.
     */
    @Select("""
            SELECT *
            FROM policies
            WHERE tenant_id = #{tenantId}
              AND workspace_id = #{workspaceId}
              AND user_id = #{userId}
            LIMIT 1
            """)
    PolicySnapshotEntity selectByScope(
            @Param("tenantId") String tenantId,
            @Param("workspaceId") String workspaceId,
            @Param("userId") String userId
    );

    /**
     * Inserts or updates one policy snapshot row by unique scope key.
     */
    @Insert("""
            INSERT INTO policies(
                tenant_id,
                workspace_id,
                user_id,
                policy_version,
                roles_json,
                denied_command_types_json,
                updated_at
            )
            VALUES(
                #{entity.tenantId},
                #{entity.workspaceId},
                #{entity.userId},
                #{entity.policyVersion},
                CAST(#{entity.rolesJson} AS jsonb),
                CAST(#{entity.deniedCommandTypesJson} AS jsonb),
                #{entity.updatedAt}
            )
            ON CONFLICT(tenant_id, workspace_id, user_id)
            DO UPDATE SET
                policy_version = EXCLUDED.policy_version,
                roles_json = EXCLUDED.roles_json,
                denied_command_types_json = EXCLUDED.denied_command_types_json,
                updated_at = EXCLUDED.updated_at
            """)
    void upsert(@Param("entity") PolicySnapshotEntity entity);
}
