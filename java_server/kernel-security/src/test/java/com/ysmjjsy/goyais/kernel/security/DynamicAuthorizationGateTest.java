/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unit tests for dynamic authorization decision ordering and policy version refresh.
 */
package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.Map;
import java.util.Set;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

class DynamicAuthorizationGateTest {

    @Test
    void shouldDenyPublicVisibilityWithoutPublisherRole() {
        InMemoryPolicySnapshotProvider provider = new InMemoryPolicySnapshotProvider();
        DynamicAuthorizationGate gate = new DynamicAuthorizationGate(provider, true);

        ExecutionContext context = new ExecutionContext(
                "tenant-a",
                "workspace-a",
                "user-a",
                Set.of("member"),
                "v1",
                "trace-1"
        );
        CommandCreateRequest request = new CommandCreateRequest("asset.publish", Map.of(), Visibility.PUBLIC);

        AuthorizationDecision decision = gate.authorize(request, context);

        Assertions.assertFalse(decision.allowed());
        Assertions.assertEquals("visibility.public.requires.publisher", decision.reason());
    }

    @Test
    void shouldReloadLatestPolicyWhenRequestPolicyVersionIsStale() {
        PolicySnapshot snapshot = new PolicySnapshot(
                "tenant-a",
                "workspace-a",
                "user-a",
                "v9",
                Set.of("member"),
                Set.of("asset.delete"),
                Instant.now()
        );
        PolicySnapshotProvider provider = new PolicySnapshotProvider() {
            /**
             * Returns the same durable snapshot for each lookup.
             */
            @Override
            public PolicySnapshot loadLatest(ExecutionContext context) {
                return snapshot;
            }

            /**
             * Simulates cache eviction while preserving durable source data.
             */
            @Override
            public void evict(String tenantId, String workspaceId, String userId) {
                // Simulates Redis eviction plus durable-store re-fetch semantics.
            }
        };

        DynamicAuthorizationGate gate = new DynamicAuthorizationGate(provider, true);
        ExecutionContext context = new ExecutionContext(
                "tenant-a",
                "workspace-a",
                "user-a",
                Set.of("member"),
                "v1",
                "trace-1"
        );
        CommandCreateRequest request = new CommandCreateRequest("asset.delete", Map.of(), Visibility.PRIVATE);

        AuthorizationDecision decision = gate.authorize(request, context);

        Assertions.assertFalse(decision.allowed());
        Assertions.assertEquals("rbac.command.blocked", decision.reason());
    }

    @Test
    void shouldApplyAclDecisionBeforeRbacAndEgress() {
        InMemoryPolicySnapshotProvider provider = new InMemoryPolicySnapshotProvider();
        DynamicAuthorizationGate gate = new DynamicAuthorizationGate(provider, true);

        ExecutionContext context = new ExecutionContext(
                "tenant-a",
                "workspace-a",
                "user-a",
                Set.of("publisher"),
                "v1",
                "trace-1"
        );
        CommandCreateRequest request = new CommandCreateRequest(
                "asset.update",
                Map.of("aclDecision", "deny"),
                Visibility.WORKSPACE
        );

        AuthorizationDecision decision = gate.authorize(request, context);

        Assertions.assertFalse(decision.allowed());
        Assertions.assertEquals("acl.command.denied", decision.reason());
    }
}
