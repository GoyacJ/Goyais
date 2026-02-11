/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Resolver that builds SQL predicates for row-level data permission.
 */
package com.ysmjjsy.goyais.kernel.mybatis;

public interface DataPermissionResolver {
    String resolveReadPredicate(String tableAlias, DataPermissionContext context);
}
