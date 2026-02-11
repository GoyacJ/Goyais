/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: In-memory fallback policy snapshot provider for minimal runtime.
 */
package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

/**
 * Keeps policy snapshots in process memory when distributed cache is unavailable.
 */
public final class InMemoryPolicySnapshotProvider implements PolicySnapshotProvider {
    private final ConcurrentMap<String, PolicySnapshot> snapshots = new ConcurrentHashMap<>();

    /**
     * Builds or refreshes one snapshot from request context defaults.
     */
    @Override
    public PolicySnapshot loadLatest(ExecutionContext context) {
        String key = scopeKey(context.tenantId(), context.workspaceId(), context.userId());
        return snapshots.compute(key, (ignored, existing) -> merge(existing, context));
    }

    /**
     * Removes one snapshot so subsequent requests must reload effective policy.
     */
    @Override
    public void evict(String tenantId, String workspaceId, String userId) {
        snapshots.remove(scopeKey(tenantId, workspaceId, userId));
    }

    /**
     * Upserts one externally loaded snapshot for the same scope.
     */
    public void upsert(PolicySnapshot snapshot) {
        snapshots.put(scopeKey(snapshot.tenantId(), snapshot.workspaceId(), snapshot.userId()), snapshot);
    }

    private PolicySnapshot merge(PolicySnapshot existing, ExecutionContext context) {
        String requestVersion = normalizeVersion(context.policyVersion());
        if (existing == null) {
            return new PolicySnapshot(
                    context.tenantId(),
                    context.workspaceId(),
                    context.userId(),
                    requestVersion,
                    context.roles(),
                    Set.of(),
                    Instant.now()
            );
        }

        if (compareVersion(existing.policyVersion(), requestVersion) >= 0) {
            return new PolicySnapshot(
                    existing.tenantId(),
                    existing.workspaceId(),
                    existing.userId(),
                    existing.policyVersion(),
                    mergeRoles(existing.roles(), context.roles()),
                    existing.deniedCommandTypes(),
                    Instant.now()
            );
        }

        return new PolicySnapshot(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                requestVersion,
                mergeRoles(existing.roles(), context.roles()),
                existing.deniedCommandTypes(),
                Instant.now()
        );
    }

    private Set<String> mergeRoles(Set<String> left, Set<String> right) {
        java.util.HashSet<String> merged = new java.util.HashSet<>();
        if (left != null) {
            merged.addAll(left);
        }
        if (right != null) {
            merged.addAll(right);
        }
        return Set.copyOf(merged);
    }

    private int compareVersion(String left, String right) {
        String l = normalizeVersion(left);
        String r = normalizeVersion(right);
        if (Objects.equals(l, r)) {
            return 0;
        }
        return l.compareTo(r);
    }

    private String normalizeVersion(String version) {
        return version == null || version.isBlank() ? "v0.1" : version.trim();
    }

    private String scopeKey(String tenantId, String workspaceId, String userId) {
        return String.join("::", safe(tenantId), safe(workspaceId), safe(userId));
    }

    private String safe(String value) {
        return value == null ? "" : value;
    }
}
