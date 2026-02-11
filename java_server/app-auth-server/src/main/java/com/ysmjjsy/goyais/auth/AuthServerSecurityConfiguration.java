/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Authorization server capability wiring reused by single app runtime.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.auth;

import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.RSAKey;
import com.nimbusds.jose.jwk.source.JWKSource;
import com.nimbusds.jose.proc.SecurityContext;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.crypto.factory.PasswordEncoderFactories;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.oauth2.core.AuthorizationGrantType;
import org.springframework.security.oauth2.core.ClientAuthenticationMethod;
import org.springframework.security.oauth2.core.oidc.OidcScopes;
import org.springframework.security.oauth2.server.authorization.client.InMemoryRegisteredClientRepository;
import org.springframework.security.oauth2.server.authorization.client.RegisteredClient;
import org.springframework.security.oauth2.server.authorization.client.RegisteredClientRepository;
import org.springframework.security.oauth2.server.authorization.settings.AuthorizationServerSettings;
import org.springframework.security.provisioning.InMemoryUserDetailsManager;
import org.springframework.security.provisioning.UserDetailsManager;
import org.springframework.security.web.SecurityFilterChain;

/**
 * Contributes OAuth2.1 and OIDC endpoints when topology mode is single.
 */
@Configuration
@ConditionalOnProperty(name = "goyais.security.topology-mode", havingValue = "single", matchIfMissing = true)
public class AuthServerSecurityConfiguration {

    /**
     * Enables OAuth2/OIDC authorization server endpoints with high precedence.
     * @param http TODO
     * @return TODO
     * @throws Exception TODO
     */
    @Bean
    @Order(Ordered.HIGHEST_PRECEDENCE)
    public SecurityFilterChain authorizationServerSecurityFilterChain(HttpSecurity http) throws Exception {
        http
                // Restrict this chain to OAuth/OIDC endpoints to avoid conflicting with API chain.
                .securityMatcher("/.well-known/**", "/oauth2/**", "/connect/**")
                // Register standard OAuth2.1/OIDC endpoint filters in single-topology mode.
                .oauth2AuthorizationServer(Customizer.withDefaults())
                .csrf(csrf -> csrf.disable())
                .authorizeHttpRequests(authorize -> authorize
                        .requestMatchers("/.well-known/**", "/oauth2/**", "/connect/**").permitAll()
                        .anyRequest().authenticated()
                )
                .formLogin(Customizer.withDefaults());
        return http.build();
    }

    /**
     * Registers bootstrap OAuth client used by vue_web in local integration mode.
     * @param passwordEncoder TODO
     * @return TODO
     */
    @Bean
    public RegisteredClientRepository registeredClientRepository(PasswordEncoder passwordEncoder) {
        RegisteredClient vueWebClient = RegisteredClient.withId(UUID.randomUUID().toString())
                .clientId("goyais-vue-web")
                .clientSecret(passwordEncoder.encode("goyais-vue-web-secret"))
                .scope("api.read")
                .scope("api.write")
                .scope(OidcScopes.OPENID)
                .scope(OidcScopes.PROFILE)
                .authorizationGrantType(AuthorizationGrantType.AUTHORIZATION_CODE)
                .authorizationGrantType(AuthorizationGrantType.REFRESH_TOKEN)
                .clientAuthenticationMethod(ClientAuthenticationMethod.CLIENT_SECRET_BASIC)
                .redirectUri("http://127.0.0.1:5173/callback")
                .build();
        return new InMemoryRegisteredClientRepository(vueWebClient);
    }

    /**
     * Publishes issuer metadata consumed by resource servers and OIDC clients.
     * @param issuer TODO
     * @return TODO
     */
    @Bean
    public AuthorizationServerSettings authorizationServerSettings(
            @Value("${goyais.security.auth-server.issuer:http://127.0.0.1:18080}") String issuer
    ) {
        return AuthorizationServerSettings.builder().issuer(issuer).build();
    }

    /**
     * Creates JWK source used by token signing and key discovery endpoints.
     * @return TODO
     */
    @Bean
    public JWKSource<SecurityContext> jwkSource() {
        RSAKey rsaKey = generateRsaKey();
        JWKSet jwkSet = new JWKSet(rsaKey);
        return (jwkSelector, securityContext) -> jwkSelector.select(jwkSet);
    }

    /**
     * Provides local users for password login and integration tests.
     * @param passwordEncoder TODO
     * @return TODO
     */
    @Bean
    public UserDetailsManager userDetailsManager(PasswordEncoder passwordEncoder) {
        return new InMemoryUserDetailsManager(
                User.withUsername("admin")
                        .password(passwordEncoder.encode("admin123"))
                        .roles("admin", "publisher")
                        .build(),
                User.withUsername("member")
                        .password(passwordEncoder.encode("member123"))
                        .roles("member")
                        .build()
        );
    }

    /**
     * Uses delegating encoder to allow future hash algorithm migrations.
     * @return TODO
     */
    @Bean
    public PasswordEncoder passwordEncoder() {
        return PasswordEncoderFactories.createDelegatingPasswordEncoder();
    }

    /**
     * Generates one ephemeral RSA key for development-token signing.
     * @return TODO
     */
    public static RSAKey generateRsaKey() {
        KeyPair keyPair = generateRsaKeyPair();
        RSAPublicKey publicKey = (RSAPublicKey) keyPair.getPublic();
        RSAPrivateKey privateKey = (RSAPrivateKey) keyPair.getPrivate();
        return new RSAKey.Builder(publicKey)
                .privateKey(privateKey)
                .keyID(UUID.randomUUID().toString())
                .build();
    }

    /**
     * Creates a 2048-bit RSA key pair used by {@link #generateRsaKey()}.
     * @return TODO
     */
    public static KeyPair generateRsaKeyPair() {
        try {
            KeyPairGenerator keyPairGenerator = KeyPairGenerator.getInstance("RSA");
            keyPairGenerator.initialize(2048);
            return keyPairGenerator.generateKeyPair();
        } catch (Exception ex) {
            throw new IllegalStateException("failed to generate RSA key pair", ex);
        }
    }
}
