/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Resolver that builds SQL predicates for row-level data permission.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
