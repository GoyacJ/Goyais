/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Resolver that builds SQL predicates for row-level data permission.
 */
package com.ysmjjsy.goyais.kernel.mybatis;

/**
 * Resolves SQL predicate fragments that enforce row-level data permission.
 */
public interface DataPermissionResolver {

    /**
     * Builds read predicate using tenant/workspace/visibility/ACL semantics.
     */
    String resolveReadPredicate(String tableAlias, DataPermissionContext context);
}
