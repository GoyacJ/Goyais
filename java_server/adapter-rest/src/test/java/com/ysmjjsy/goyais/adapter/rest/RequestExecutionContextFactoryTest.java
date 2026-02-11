/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unit tests for execution context resolution from JWT and dev headers.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.util.List;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;

class RequestExecutionContextFactoryTest {

    @Test
    void shouldResolveExecutionContextFromJwtClaims() {
        RequestExecutionContextFactory factory = new RequestExecutionContextFactory(false);
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.addHeader("X-Trace-Id", "trace-1");

        Jwt jwt = Jwt.withTokenValue("token")
                .header("alg", "none")
                .claim("tenantId", "tenant-a")
                .claim("workspaceId", "workspace-a")
                .claim("userId", "user-a")
                .claim("roles", List.of("member", "publisher"))
                .claim("policyVersion", "v9")
                .build();

        ExecutionContext context = factory.resolve(new JwtAuthenticationToken(jwt), request);

        Assertions.assertEquals("tenant-a", context.tenantId());
        Assertions.assertEquals("workspace-a", context.workspaceId());
        Assertions.assertEquals("user-a", context.userId());
        Assertions.assertTrue(context.roles().contains("publisher"));
        Assertions.assertEquals("v9", context.policyVersion());
        Assertions.assertEquals("trace-1", context.traceId());
    }

    @Test
    void shouldFallbackToHeadersWhenDevModeEnabled() {
        RequestExecutionContextFactory factory = new RequestExecutionContextFactory(true);
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.addHeader("X-Tenant-Id", "tenant-a");
        request.addHeader("X-Workspace-Id", "workspace-a");
        request.addHeader("X-User-Id", "user-a");
        request.addHeader("X-Roles", "member,publisher");
        request.addHeader("X-Policy-Version", "v2");
        request.addHeader("X-Trace-Id", "trace-2");

        ExecutionContext context = factory.resolve(null, request);

        Assertions.assertEquals("tenant-a", context.tenantId());
        Assertions.assertEquals("workspace-a", context.workspaceId());
        Assertions.assertEquals("user-a", context.userId());
        Assertions.assertTrue(context.roles().contains("publisher"));
        Assertions.assertEquals("v2", context.policyVersion());
        Assertions.assertEquals("trace-2", context.traceId());
    }

    @Test
    void shouldFailWhenNeitherJwtNorHeadersAreAvailable() {
        RequestExecutionContextFactory factory = new RequestExecutionContextFactory(false);

        IllegalStateException ex = Assertions.assertThrows(
                IllegalStateException.class,
                () -> factory.resolve(null, new MockHttpServletRequest())
        );

        Assertions.assertTrue(ex.getMessage().contains("missing authenticated execution context"));
    }
}
