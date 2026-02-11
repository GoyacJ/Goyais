/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Integration tests for single-topology security behavior and OAuth endpoint exposure.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.api;

import com.ysmjjsy.goyais.adapter.rest.GlobalExceptionHandler;
import com.ysmjjsy.goyais.adapter.rest.HealthController;
import com.ysmjjsy.goyais.auth.AuthServerSecurityConfiguration;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringBootConfiguration;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest(
        classes = ApiSecuritySingleModeIntegrationTest.TestApplication.class,
        properties = {
                "goyais.security.topology-mode=single",
                "spring.autoconfigure.exclude=org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration,"
                        + "org.springframework.boot.autoconfigure.flyway.FlywayAutoConfiguration,"
                        + "com.baomidou.mybatisplus.autoconfigure.MybatisPlusAutoConfiguration"
        }
)
@AutoConfigureMockMvc
class ApiSecuritySingleModeIntegrationTest {
    @Autowired
    private MockMvc mockMvc;

    @Test
    void shouldExposeHealthzWithoutAuthentication() throws Exception {
        mockMvc.perform(get("/api/v1/healthz"))
                .andExpect(status().isOk());
    }

    @Test
    void shouldExposeOAuthJwksEndpointInSingleMode() throws Exception {
        mockMvc.perform(get("/oauth2/jwks"))
                .andExpect(status().isOk());
    }

    @Test
    void shouldRequireBearerTokenForProtectedApi() throws Exception {
        mockMvc.perform(get("/api/v1/test-secure/ok"))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void shouldReturnForbiddenEnvelopeWhenAuthenticatedRequestIsDenied() throws Exception {
        mockMvc.perform(get("/api/v1/test-secure/deny")
                        .with(SecurityMockMvcRequestPostProcessors.jwt().jwt(jwt -> jwt
                                .claim("tenantId", "tenant-a")
                                .claim("workspaceId", "workspace-a")
                                .claim("userId", "user-a")
                                .claim("policyVersion", "v1")
                                .claim("roles", java.util.List.of("member"))
                        )))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error.code").value("FORBIDDEN"))
                .andExpect(jsonPath("$.error.messageKey").value("error.authz.forbidden"))
                .andExpect(jsonPath("$.error.details.reason").value("permission_denied_by_test"));
    }

    @Test
    void shouldReturnNotFoundEnvelopeForMissingRoute() throws Exception {
        mockMvc.perform(get("/api/v1/test-secure/missing")
                        .with(SecurityMockMvcRequestPostProcessors.jwt().jwt(jwt -> jwt
                                .claim("tenantId", "tenant-a")
                                .claim("workspaceId", "workspace-a")
                                .claim("userId", "user-a")
                                .claim("policyVersion", "v1")
                                .claim("roles", java.util.List.of("member"))
                        )))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.error.code").value("NOT_FOUND"))
                .andExpect(jsonPath("$.error.messageKey").value("error.request.not_found"));
    }

    @SpringBootConfiguration
    @EnableAutoConfiguration
    @Import({
            ApiSecurityConfiguration.class,
            AuthServerSecurityConfiguration.class,
            HealthController.class,
            GlobalExceptionHandler.class,
            ProtectedTestController.class
    })
    static class TestApplication {
    }

    /**
     * Test-only protected endpoint used to verify authenticated deny path mapping.
     */
    @RestController
    @RequestMapping("/api/v1/test-secure")
    static class ProtectedTestController {

        /**
         * Returns a simple payload for authenticated requests.
         */
        @GetMapping(value = "/ok", produces = MediaType.APPLICATION_JSON_VALUE)
        Map<String, Object> ok() {
            return Map.of("status", "ok");
        }

        /**
         * Throws authorization-style error to verify error-envelope mapping.
         */
        @GetMapping(value = "/deny", produces = MediaType.APPLICATION_JSON_VALUE)
        Map<String, Object> deny() {
            throw new IllegalStateException("permission_denied_by_test");
        }
    }
}
