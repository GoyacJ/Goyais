/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: API application wiring for dynamic authz, command pipeline, and cache invalidation.
 */
package com.ysmjjsy.goyais.api;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.command.CommandPipeline;
import com.ysmjjsy.goyais.capability.cache.policy.RedisPolicyInvalidationPublisher;
import com.ysmjjsy.goyais.capability.cache.policy.RedisPolicyInvalidationSubscriber;
import com.ysmjjsy.goyais.capability.event.DomainEvent;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionResolver;
import com.ysmjjsy.goyais.kernel.mybatis.DefaultDataPermissionResolver;
import com.ysmjjsy.goyais.kernel.security.AuthorizationDecision;
import com.ysmjjsy.goyais.kernel.security.AuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.DynamicAuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.EgressGate;
import com.ysmjjsy.goyais.kernel.security.InMemoryPolicyInvalidationBus;
import com.ysmjjsy.goyais.kernel.security.InMemoryPolicySnapshotProvider;
import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationEvent;
import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationPublisher;
import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationSubscriber;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.ApplicationRunner;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;

/**
 * Provides default runtime beans for single-app and multi-resource-server topologies.
 */
@Configuration
public class ApiServerConfiguration {

    /**
     * Provides in-memory policy snapshots and acts as Redis-failure fallback cache.
     */
    @Bean
    public InMemoryPolicySnapshotProvider policySnapshotProvider() {
        return new InMemoryPolicySnapshotProvider();
    }

    /**
     * Provides local invalidation bus used when Redis pubsub is unavailable.
     */
    @Bean
    public InMemoryPolicyInvalidationBus inMemoryPolicyInvalidationBus() {
        return new InMemoryPolicyInvalidationBus();
    }

    /**
     * Selects Redis or in-memory publisher for policy invalidation fan-out.
     */
    @Bean
    public PolicyInvalidationPublisher policyInvalidationPublisher(
            ObjectProvider<StringRedisTemplate> redisTemplateProvider,
            InMemoryPolicyInvalidationBus inMemoryBus,
            @Value("${goyais.security.authz.policy-invalidation-channel:goyais:policy:invalidate}") String channel
    ) {
        StringRedisTemplate redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return inMemoryBus;
        }
        return new RedisPolicyInvalidationPublisher(redisTemplate, channel);
    }

    /**
     * Selects Redis or in-memory subscriber to receive policy invalidation events.
     */
    @Bean
    public PolicyInvalidationSubscriber policyInvalidationSubscriber(
            ObjectProvider<RedisMessageListenerContainer> listenerContainerProvider,
            InMemoryPolicyInvalidationBus inMemoryBus,
            @Value("${goyais.security.authz.policy-invalidation-channel:goyais:policy:invalidate}") String channel
    ) {
        RedisMessageListenerContainer listenerContainer = listenerContainerProvider.getIfAvailable();
        if (listenerContainer == null) {
            return inMemoryBus;
        }
        return new RedisPolicyInvalidationSubscriber(listenerContainer, channel);
    }

    /**
     * Starts invalidation listener so remote policy updates evict local snapshot cache.
     */
    @Bean
    public ApplicationRunner policyInvalidationListener(
            PolicyInvalidationSubscriber subscriber,
            InMemoryPolicySnapshotProvider snapshotProvider
    ) {
        return args -> subscriber.start(event -> snapshotProvider.evict(
                event.tenantId(),
                event.workspaceId(),
                event.userId()
        ));
    }

    /**
     * Creates policyVersion-aware authorization gate with runtime enable switch.
     */
    @Bean
    public AuthorizationGate authorizationGate(
            InMemoryPolicySnapshotProvider snapshotProvider,
            @Value("${goyais.security.authz.dynamic-enabled:true}") boolean dynamicEnabled
    ) {
        return new DynamicAuthorizationGate(snapshotProvider, dynamicEnabled);
    }

    /**
     * Enforces minimal egress control and blocks explicit deny policy in payload.
     */
    @Bean
    public EgressGate egressGate() {
        return (request, context) -> {
            Object rawPolicy = request.payload() == null ? null : request.payload().get("egressPolicy");
            if (rawPolicy != null && "deny".equalsIgnoreCase(String.valueOf(rawPolicy))) {
                return AuthorizationDecision.deny("egress.policy.denied");
            }
            return AuthorizationDecision.allow("egress.policy.allowed");
        };
    }

    /**
     * Uses in-process sink while event bus integration is still in bootstrap phase.
     */
    @Bean
    public DomainEventPublisher domainEventPublisher() {
        return event -> {
            // Intentionally no-op until outbox/kafka provider is enabled.
        };
    }

    /**
     * Provides default command handler and emits policy invalidation on policy refresh command.
     */
    @Bean
    public CommandHandler defaultCommandHandler(
            PolicyInvalidationPublisher invalidationPublisher,
            InMemoryPolicySnapshotProvider snapshotProvider
    ) {
        return new CommandHandler() {
            /**
             * Accepts all commands during bootstrap to preserve command-first compatibility.
             */
            @Override
            public boolean supports(String commandType) {
                return true;
            }

            /**
             * Executes command and broadcasts policy invalidation when policy is refreshed.
             */
            @Override
            public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
                if ("policy.refresh".equals(request.commandType())) {
                    String newVersion = request.payload() == null || request.payload().get("policyVersion") == null
                            ? context.policyVersion()
                            : String.valueOf(request.payload().get("policyVersion"));
                    PolicyInvalidationEvent event = new PolicyInvalidationEvent(
                            context.tenantId(),
                            context.workspaceId(),
                            context.userId(),
                            newVersion,
                            context.traceId(),
                            Instant.now()
                    );
                    snapshotProvider.evict(context.tenantId(), context.workspaceId(), context.userId());
                    invalidationPublisher.publish(event);
                    return Map.of(
                            "commandType", request.commandType(),
                            "status", "policy-invalidated",
                            "policyVersion", newVersion
                    );
                }

                return Map.of(
                        "commandType", request.commandType(),
                        "tenantId", context.tenantId(),
                        "workspaceId", context.workspaceId(),
                        "policyVersion", context.policyVersion(),
                        "status", "executed"
                );
            }
        };
    }

    /**
     * Provides default row-level data permission resolver for MyBatis integration.
     */
    @Bean
    public DataPermissionResolver dataPermissionResolver() {
        return new DefaultDataPermissionResolver();
    }

    /**
     * Composes command pipeline with authorization, egress and event publication.
     */
    @Bean
    public CommandPipeline commandPipeline(
            AuthorizationGate authorizationGate,
            EgressGate egressGate,
            List<CommandHandler> handlers,
            DomainEventPublisher eventPublisher
    ) {
        return new CommandPipeline(authorizationGate, egressGate, handlers, eventPublisher);
    }
}
