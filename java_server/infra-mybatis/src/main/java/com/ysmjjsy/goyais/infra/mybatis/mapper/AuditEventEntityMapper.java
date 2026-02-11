/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Mapper for audit event persistence.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.AuditEventEntity;

/**
 * Persists command authorization and execution audit rows.
 */
public interface AuditEventEntityMapper extends BaseMapper<AuditEventEntity> {
}
