/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Security bootstrap for API server with open endpoints in design phase.
 */
package com.ysmjjsy.goyais.api;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.annotation.Order;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.web.SecurityFilterChain;

/**
 * Configures resource-server-facing security rules for API endpoints.
 */
@Configuration
public class ApiSecurityConfiguration {

    /**
     * Applies default API security while supporting single and resource-only topology modes.
     */
    @Bean
    @Order(200)
    public SecurityFilterChain apiSecurityFilterChain(
            HttpSecurity http,
            @Value("${goyais.security.topology-mode:single}") String topologyMode
    ) throws Exception {
        http
                .csrf(csrf -> csrf.disable())
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers("/api/v1/healthz", "/api/v1/system/healthz", "/actuator/**").permitAll()
                        .requestMatchers("/oauth2/jwks", "/oauth2/token", "/oauth2/authorize", "/.well-known/**")
                        .access((authentication, context) -> new org.springframework.security.authorization.AuthorizationDecision(
                                !"resource-only".equalsIgnoreCase(topologyMode)
                        ))
                        .anyRequest().permitAll()
                )
                .httpBasic(Customizer.withDefaults());
        return http.build();
    }
}
