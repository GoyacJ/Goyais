/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Default SQL predicate resolver for row-level data permission.
 */
package com.ysmjjsy.goyais.kernel.mybatis;

/**
 * Generates SQL fragments aligned with owner/workspace/ACL read semantics.
 */
public final class DefaultDataPermissionResolver implements DataPermissionResolver {

    /**
     * Returns one deterministic predicate with policyVersion marker for auditability.
     */
    @Override
    public String resolveReadPredicate(String tableAlias, DataPermissionContext context) {
        String alias = tableAlias == null || tableAlias.isBlank() ? "" : tableAlias.trim() + ".";
        String safePolicyVersion = context.policyVersion() == null ? "v0.1" : context.policyVersion();
        return "("
                + alias + "tenant_id = #{dp.tenantId}"
                + " AND " + alias + "workspace_id = #{dp.workspaceId}"
                + " AND ("
                + alias + "owner_id = #{dp.userId}"
                + " OR " + alias + "visibility = 'WORKSPACE'"
                + " OR EXISTS (SELECT 1 FROM acl_entries a"
                + " WHERE a.resource_id = " + alias + "id"
                + " AND a.resource_type = 'command'"
                + " AND a.subject_id = #{dp.userId}"
                + " AND a.permissions::text LIKE '%READ%')"
                + ")) /* policyVersion=" + safePolicyVersion + " */";
    }
}
