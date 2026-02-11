/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Builds ExecutionContext from JWT claims with optional dev header fallback.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.util.Arrays;
import java.util.Collection;
import java.util.LinkedHashSet;
import java.util.Set;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;

/**
 * Resolves agent-as-user context from authenticated principal and dev-only fallback headers.
 */
@Component
public final class RequestExecutionContextFactory {
    private final boolean devHeaderContextEnabled;

    /**
     * Creates context resolver with runtime guard for dev header fallback behavior.
     * @param devHeaderContextEnabled TODO
     * @return TODO
     */
    public RequestExecutionContextFactory(
            @Value("${goyais.security.dev-header-context-enabled:false}") boolean devHeaderContextEnabled
    ) {
        this.devHeaderContextEnabled = devHeaderContextEnabled;
    }

    /**
     * Resolves execution context from JWT claims and optional X-* headers in dev mode.
     * @param authentication TODO
     * @param request TODO
     * @return TODO
     */
    public ExecutionContext resolve(Authentication authentication, HttpServletRequest request) {
        if (authentication instanceof JwtAuthenticationToken jwtAuthenticationToken) {
            return fromJwt(jwtAuthenticationToken.getToken(), request);
        }

        if (devHeaderContextEnabled) {
            ExecutionContext headerContext = fromHeaders(request);
            if (headerContext != null) {
                return headerContext;
            }
        }

        throw new IllegalStateException("missing authenticated execution context");
    }

    private ExecutionContext fromJwt(Jwt jwt, HttpServletRequest request) {
        String tenantId = stringClaim(jwt, "tenantId", "tenant_id");
        String workspaceId = stringClaim(jwt, "workspaceId", "workspace_id");
        String userId = stringClaim(jwt, "userId", "sub", "user_id");
        String policyVersion = stringClaim(jwt, "policyVersion", "policy_version");
        Set<String> roles = roleClaims(jwt);
        String traceId = headerOrGenerated(request, "X-Trace-Id");

        if (isBlank(tenantId) || isBlank(workspaceId) || isBlank(userId)) {
            if (!devHeaderContextEnabled) {
                throw new IllegalStateException("jwt claim missing tenant/workspace/user");
            }
            ExecutionContext fallback = fromHeaders(request);
            if (fallback != null) {
                return fallback;
            }
            throw new IllegalStateException("jwt claim missing tenant/workspace/user");
        }

        return new ExecutionContext(
                tenantId,
                workspaceId,
                userId,
                roles,
                isBlank(policyVersion) ? "v0.1" : policyVersion,
                traceId
        );
    }

    private ExecutionContext fromHeaders(HttpServletRequest request) {
        String tenantId = request.getHeader("X-Tenant-Id");
        String workspaceId = request.getHeader("X-Workspace-Id");
        String userId = request.getHeader("X-User-Id");
        if (isBlank(tenantId) || isBlank(workspaceId) || isBlank(userId)) {
            return null;
        }

        return new ExecutionContext(
                tenantId,
                workspaceId,
                userId,
                splitRoles(request.getHeader("X-Roles")),
                defaultIfBlank(request.getHeader("X-Policy-Version"), "v0.1"),
                headerOrGenerated(request, "X-Trace-Id")
        );
    }

    private String stringClaim(Jwt jwt, String... claimNames) {
        for (String claimName : claimNames) {
            Object value = jwt.getClaims().get(claimName);
            if (value != null && !String.valueOf(value).isBlank()) {
                return String.valueOf(value);
            }
        }
        return null;
    }

    private Set<String> roleClaims(Jwt jwt) {
        Set<String> roles = new LinkedHashSet<>();

        Object directRoles = jwt.getClaims().get("roles");
        if (directRoles instanceof Collection<?> collection) {
            for (Object item : collection) {
                if (item != null && !String.valueOf(item).isBlank()) {
                    roles.add(String.valueOf(item));
                }
            }
        }

        Object scope = jwt.getClaims().get("scope");
        if (scope != null) {
            roles.addAll(splitRoles(String.valueOf(scope)));
        }

        if (roles.isEmpty()) {
            roles.add("member");
        }

        return Set.copyOf(roles);
    }

    private Set<String> splitRoles(String value) {
        if (isBlank(value)) {
            return Set.of("member");
        }
        Set<String> roles = Arrays.stream(value.split("[, ]"))
                .map(String::trim)
                .filter(item -> !item.isBlank())
                .collect(java.util.stream.Collectors.toCollection(LinkedHashSet::new));
        return roles.isEmpty() ? Set.of("member") : Set.copyOf(roles);
    }

    private String headerOrGenerated(HttpServletRequest request, String headerName) {
        return defaultIfBlank(request.getHeader(headerName), UUID.randomUUID().toString());
    }

    private String defaultIfBlank(String value, String defaultValue) {
        return isBlank(value) ? defaultValue : value.trim();
    }

    private boolean isBlank(String value) {
        return value == null || value.isBlank();
    }
}
