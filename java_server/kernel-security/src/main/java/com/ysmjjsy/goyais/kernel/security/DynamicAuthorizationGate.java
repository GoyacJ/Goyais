/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Dynamic authorization gate backed by policyVersion-aware snapshots.
 */
package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;

/**
 * Enforces tenant/workspace/visibility/RBAC guard checks using latest policy snapshot.
 */
public final class DynamicAuthorizationGate implements AuthorizationGate {
    private final PolicySnapshotProvider snapshotProvider;
    private final boolean dynamicEnabled;

    /**
     * Creates a dynamic gate that can be disabled by runtime configuration.
     */
    public DynamicAuthorizationGate(PolicySnapshotProvider snapshotProvider, boolean dynamicEnabled) {
        this.snapshotProvider = snapshotProvider;
        this.dynamicEnabled = dynamicEnabled;
    }

    /**
     * Evaluates one command request against the latest effective policy snapshot.
     */
    @Override
    public AuthorizationDecision authorize(CommandCreateRequest request, ExecutionContext context) {
        if (!dynamicEnabled) {
            return AuthorizationDecision.allow("dynamic-authz-disabled");
        }

        PolicySnapshot snapshot = snapshotProvider.loadLatest(context);
        PolicyVersionState versionState = evaluateVersionState(context.policyVersion(), snapshot.policyVersion());
        if (versionState.stale()) {
            snapshotProvider.evict(context.tenantId(), context.workspaceId(), context.userId());
            snapshot = snapshotProvider.loadLatest(context);
            versionState = evaluateVersionState(context.policyVersion(), snapshot.policyVersion());
        }

        AuthorizationDecision tenantDecision = checkTenant(snapshot, context);
        if (!tenantDecision.allowed()) {
            return tenantDecision;
        }

        AuthorizationDecision workspaceDecision = checkWorkspace(snapshot, context);
        if (!workspaceDecision.allowed()) {
            return workspaceDecision;
        }

        AuthorizationDecision visibilityDecision = checkVisibility(request, snapshot);
        if (!visibilityDecision.allowed()) {
            return visibilityDecision;
        }

        AuthorizationDecision rbacDecision = checkRbac(request, snapshot);
        if (!rbacDecision.allowed()) {
            return rbacDecision;
        }

        return AuthorizationDecision.allow("policyVersion=" + versionState.effectiveVersion());
    }

    private AuthorizationDecision checkTenant(PolicySnapshot snapshot, ExecutionContext context) {
        if (snapshot.tenantId() == null || snapshot.tenantId().isBlank()) {
            return AuthorizationDecision.deny("tenant.missing");
        }
        if (!snapshot.tenantId().equals(context.tenantId())) {
            return AuthorizationDecision.deny("tenant.mismatch");
        }
        return AuthorizationDecision.allow("tenant.allow");
    }

    private AuthorizationDecision checkWorkspace(PolicySnapshot snapshot, ExecutionContext context) {
        if (snapshot.workspaceId() == null || snapshot.workspaceId().isBlank()) {
            return AuthorizationDecision.deny("workspace.missing");
        }
        if (!snapshot.workspaceId().equals(context.workspaceId())) {
            return AuthorizationDecision.deny("workspace.mismatch");
        }
        return AuthorizationDecision.allow("workspace.allow");
    }

    private AuthorizationDecision checkVisibility(CommandCreateRequest request, PolicySnapshot snapshot) {
        Visibility visibility = request.visibility() == null ? Visibility.PRIVATE : request.visibility();
        if (visibility == Visibility.PUBLIC && !hasAnyRole(snapshot, "admin", "publisher")) {
            return AuthorizationDecision.deny("visibility.public.requires.publisher");
        }
        return AuthorizationDecision.allow("visibility.allow");
    }

    private AuthorizationDecision checkRbac(CommandCreateRequest request, PolicySnapshot snapshot) {
        if (snapshot.deniedCommandTypes().contains(request.commandType())) {
            return AuthorizationDecision.deny("rbac.command.blocked");
        }
        return AuthorizationDecision.allow("rbac.allow");
    }

    private PolicyVersionState evaluateVersionState(String requestVersion, String effectiveVersion) {
        String normalizedRequestVersion = normalizeVersion(requestVersion);
        String normalizedEffectiveVersion = normalizeVersion(effectiveVersion);
        boolean stale = normalizedRequestVersion.compareTo(normalizedEffectiveVersion) < 0;
        return new PolicyVersionState(
                normalizedRequestVersion,
                normalizedEffectiveVersion,
                stale,
                Instant.now()
        );
    }

    private boolean hasAnyRole(PolicySnapshot snapshot, String... expected) {
        for (String role : expected) {
            if (snapshot.roles().contains(role)) {
                return true;
            }
        }
        return false;
    }

    private String normalizeVersion(String version) {
        return version == null || version.isBlank() ? "v0.1" : version.trim();
    }
}
