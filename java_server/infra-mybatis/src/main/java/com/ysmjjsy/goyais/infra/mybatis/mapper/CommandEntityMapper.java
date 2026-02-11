/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for command persistence and readable-query operations.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.CommandEntity;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import java.util.List;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

/**
 * Executes command row CRUD and data-permission-aware queries.
 */
public interface CommandEntityMapper extends BaseMapper<CommandEntity> {

    /**
     * Returns one readable command entity by identifier and SQL predicate.
     */
    @Select("""
            <script>
            SELECT *
            FROM commands c
            WHERE c.id = #{commandId}
              AND ${predicate}
            LIMIT 1
            </script>
            """)
    CommandEntity selectReadableById(
            @Param("commandId") String commandId,
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext
    );

    /**
     * Returns readable command entities with stable descending ordering.
     */
    @Select("""
            <script>
            SELECT *
            FROM commands c
            WHERE ${predicate}
            ORDER BY c.created_at DESC, c.id DESC
            LIMIT #{limit}
            </script>
            """)
    List<CommandEntity> selectReadableList(
            @Param("predicate") String predicate,
            @Param("dp") DataPermissionContext dataPermissionContext,
            @Param("limit") int limit
    );
}
