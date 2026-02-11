/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>API security enforcing bearer-token auth for business endpoints.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.api;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.annotation.Order;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;

/**
 * Configures resource-server-facing security rules for API endpoints.
 */
@Configuration
public class ApiSecurityConfiguration {

    /**
     * Applies API security while supporting single and resource-only topology modes.
     * @param http TODO
     * @param topologyMode TODO
     * @return TODO
     */
    @Bean
    @Order(200)
    public SecurityFilterChain apiSecurityFilterChain(
            HttpSecurity http,
            @Value("${goyais.security.topology-mode:single}") String topologyMode
    ) throws Exception {
        boolean resourceOnly = "resource-only".equalsIgnoreCase(topologyMode);
        http
                .csrf(csrf -> csrf.disable())
                .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers("/api/v1/healthz", "/api/v1/system/healthz").permitAll()
                        .requestMatchers("/actuator/health", "/actuator/info", "/error").permitAll()
                        .requestMatchers("/oauth2/**", "/.well-known/**", "/connect/**")
                        .access((authentication, context) -> new org.springframework.security.authorization.AuthorizationDecision(
                                !resourceOnly
                        ))
                        .requestMatchers("/api/v1/**").authenticated()
                        .anyRequest().denyAll()
                )
                .oauth2ResourceServer(resourceServer -> resourceServer.jwt(Customizer.withDefaults()));
        return http.build();
    }
}
