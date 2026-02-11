/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Default SQL predicate resolver for row-level data permission.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.mybatis;

/**
 * Generates SQL fragments aligned with owner/workspace/ACL read semantics.
 */
public final class DefaultDataPermissionResolver implements DataPermissionResolver {

    /**
     * Returns one deterministic predicate with policyVersion marker for auditability.
     * @param tableAlias TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public String resolveReadPredicate(String tableAlias, DataPermissionContext context) {
        String alias = tableAlias == null || tableAlias.isBlank() ? "" : tableAlias.trim() + ".";
        String resourceType = sanitizeResourceType(context.resourceType());
        String requiredPermission = sanitizePermission(context.requiredPermission());
        String safePolicyVersion = context.policyVersion() == null ? "v0.1" : context.policyVersion();
        return "("
                + alias + "tenant_id = #{dp.tenantId}"
                + " AND " + alias + "workspace_id = #{dp.workspaceId}"
                + " AND ("
                + alias + "owner_id = #{dp.userId}"
                + " OR " + alias + "visibility = 'WORKSPACE'"
                + " OR EXISTS (SELECT 1 FROM acl_entries a"
                + " WHERE a.tenant_id = " + alias + "tenant_id"
                + " AND a.workspace_id = " + alias + "workspace_id"
                + " AND a.resource_id = " + alias + "id"
                + " AND a.resource_type = '" + resourceType + "'"
                + " AND a.subject_type = 'user'"
                + " AND a.subject_id = #{dp.userId}"
                + " AND (a.expires_at IS NULL OR a.expires_at >= CURRENT_TIMESTAMP)"
                + " AND a.permissions @> jsonb_build_array('" + requiredPermission + "'))"
                + ")) /* policyVersion=" + safePolicyVersion + " */";
    }

    private String sanitizeResourceType(String value) {
        String normalized = value == null || value.isBlank() ? "command" : value.trim().toLowerCase();
        return normalized.matches("[a-z0-9_:-]+") ? normalized : "command";
    }

    private String sanitizePermission(String value) {
        String normalized = value == null || value.isBlank() ? "READ" : value.trim().toUpperCase();
        return normalized.matches("[A-Z_]+") ? normalized : "READ";
    }
}
