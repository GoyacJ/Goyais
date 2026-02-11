/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Integration tests for resource-only topology security behavior.
 */
package com.ysmjjsy.goyais.api;

import com.ysmjjsy.goyais.adapter.rest.GlobalExceptionHandler;
import com.ysmjjsy.goyais.adapter.rest.HealthController;
import com.ysmjjsy.goyais.auth.AuthServerSecurityConfiguration;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.SpringBootConfiguration;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest(
        classes = ApiSecurityResourceOnlyModeIntegrationTest.TestApplication.class,
        properties = {
                "goyais.security.topology-mode=resource-only",
                "spring.autoconfigure.exclude=org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration,"
                        + "org.springframework.boot.autoconfigure.flyway.FlywayAutoConfiguration,"
                        + "com.baomidou.mybatisplus.autoconfigure.MybatisPlusAutoConfiguration"
        }
)
@AutoConfigureMockMvc
class ApiSecurityResourceOnlyModeIntegrationTest {
    @Autowired
    private MockMvc mockMvc;

    @Test
    void shouldExposeHealthzWithoutAuthentication() throws Exception {
        mockMvc.perform(get("/api/v1/healthz"))
                .andExpect(status().isOk());
    }

    @Test
    void shouldCloseOAuthJwksEndpointInResourceOnlyMode() throws Exception {
        mockMvc.perform(get("/oauth2/jwks"))
                .andExpect(status().is4xxClientError());
    }

    @Test
    void shouldRequireBearerTokenForProtectedApi() throws Exception {
        mockMvc.perform(get("/api/v1/test-secure/ok"))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void shouldReturnMethodNotAllowedEnvelopeForUnsupportedVerb() throws Exception {
        mockMvc.perform(post("/api/v1/healthz"))
                .andExpect(status().isMethodNotAllowed())
                .andExpect(jsonPath("$.error.code").value("METHOD_NOT_ALLOWED"))
                .andExpect(jsonPath("$.error.messageKey").value("error.request.method_not_allowed"));
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
     * Test-only protected endpoint used to verify authentication requirements.
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
    }
}
