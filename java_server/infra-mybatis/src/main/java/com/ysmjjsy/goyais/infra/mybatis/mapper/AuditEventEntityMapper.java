/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Mapper for audit event persistence.
 */
package com.ysmjjsy.goyais.infra.mybatis.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.AuditEventEntity;

/**
 * Persists command authorization and execution audit rows.
 */
public interface AuditEventEntityMapper extends BaseMapper<AuditEventEntity> {
}
