/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>API application wiring for dynamic authz, command pipeline, and cache invalidation.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.api;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.audit.AuditEventStore;
import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.command.CommandPipeline;
import com.ysmjjsy.goyais.capability.cache.policy.RedisPolicyInvalidationPublisher;
import com.ysmjjsy.goyais.capability.cache.policy.RedisPolicyInvalidationSubscriber;
import com.ysmjjsy.goyais.capability.cache.policy.RedisPolicySnapshotProvider;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.capability.storage.LocalObjectStorage;
import com.ysmjjsy.goyais.capability.storage.ObjectStorage;
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
import com.ysmjjsy.goyais.kernel.security.PolicySnapshot;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshotProvider;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshotStore;
import java.nio.file.Path;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.Set;
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
     * Provides in-memory policy snapshots as Redis-unavailable fallback cache.
     * @return TODO
     */
    @Bean
    public InMemoryPolicySnapshotProvider inMemoryPolicySnapshotProvider() {
        return new InMemoryPolicySnapshotProvider();
    }

    /**
     * Selects Redis-first provider with durable-store fallback when Redis is available.
     * @param redisTemplateProvider TODO
     * @param policySnapshotStore TODO
     * @param inMemoryPolicySnapshotProvider TODO
     * @param objectMapper TODO
     * @param cacheTtl TODO
     * @return TODO
     */
    @Bean
    public PolicySnapshotProvider policySnapshotProvider(
            ObjectProvider<StringRedisTemplate> redisTemplateProvider,
            PolicySnapshotStore policySnapshotStore,
            InMemoryPolicySnapshotProvider inMemoryPolicySnapshotProvider,
            ObjectMapper objectMapper,
            @Value("${goyais.security.authz.policy-cache-ttl:30s}") Duration cacheTtl
    ) {
        StringRedisTemplate redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return inMemoryPolicySnapshotProvider;
        }

        return new RedisPolicySnapshotProvider(redisTemplate, policySnapshotStore, cacheTtl, objectMapper);
    }

    /**
     * Provides local invalidation bus used when Redis pubsub is unavailable.
     * @return TODO
     */
    @Bean
    public InMemoryPolicyInvalidationBus inMemoryPolicyInvalidationBus() {
        return new InMemoryPolicyInvalidationBus();
    }

    /**
     * Selects Redis or in-memory publisher for policy invalidation fan-out.
     * @param redisTemplateProvider TODO
     * @param inMemoryBus TODO
     * @param channel TODO
     * @return TODO
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
     * @param listenerContainerProvider TODO
     * @param inMemoryBus TODO
     * @param channel TODO
     * @return TODO
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
     * @param subscriber TODO
     * @param snapshotProvider TODO
     * @return TODO
     */
    @Bean
    public ApplicationRunner policyInvalidationListener(
            PolicyInvalidationSubscriber subscriber,
            PolicySnapshotProvider snapshotProvider
    ) {
        return args -> subscriber.start(event -> snapshotProvider.evict(
                event.tenantId(),
                event.workspaceId(),
                event.userId()
        ));
    }

    /**
     * Creates policyVersion-aware authorization gate with runtime enable switch.
     * @param snapshotProvider TODO
     * @param dynamicEnabled TODO
     * @return TODO
     */
    @Bean
    public AuthorizationGate authorizationGate(
            PolicySnapshotProvider snapshotProvider,
            @Value("${goyais.security.authz.dynamic-enabled:true}") boolean dynamicEnabled
    ) {
        return new DynamicAuthorizationGate(snapshotProvider, dynamicEnabled);
    }

    /**
     * Enforces minimal egress control and blocks explicit deny policy in payload.
     * @return TODO
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
     * Provides local object storage provider for minimal and bootstrap profiles.
     * @param provider TODO
     * @return TODO
     */
    @Bean
    public ObjectStorage objectStorage(
            @Value("${goyais.storage.provider:local}") String provider,
            @Value("${goyais.storage.local-root:${java.io.tmpdir}/goyais-storage}") String localRoot
    ) {
        // v0.1 bootstrap maps all providers to local filesystem to keep runtime minimal.
        String normalized = provider == null ? "local" : provider.trim().toLowerCase();
        return switch (normalized) {
            case "local" -> new LocalObjectStorage(Path.of(localRoot));
            case "minio", "s3" -> new LocalObjectStorage(Path.of(localRoot).resolve(normalized));
            default -> throw new IllegalStateException("unsupported object storage provider: " + provider);
        };
    }

    /**
     * Uses in-process sink while event bus integration is still in bootstrap phase.
     * @return TODO
     */
    @Bean
    public DomainEventPublisher domainEventPublisher() {
        return event -> {
            // Intentionally no-op until outbox/kafka provider is enabled.
        };
    }

    /**
     * Provides default command handler and emits policy invalidation on policy refresh command.
     * @param invalidationPublisher TODO
     * @param snapshotProvider TODO
     * @param policySnapshotStore TODO
     * @return TODO
     */
    @Bean
    public CommandHandler defaultCommandHandler(
            PolicyInvalidationPublisher invalidationPublisher,
            PolicySnapshotProvider snapshotProvider,
            PolicySnapshotStore policySnapshotStore
    ) {
        return new CommandHandler() {
            /**
             * Accepts non-asset/non-share/non-workflow commands during bootstrap fallback.
             * @param commandType TODO
             * @return TODO
             */
            @Override
            public boolean supports(String commandType) {
                if (commandType == null || commandType.isBlank()) {
                    return false;
                }
                if (commandType.startsWith("asset.")
                        || commandType.startsWith("share.")
                        || commandType.startsWith("workflow.")) {
                    return false;
                }
                return true;
            }

            /**
             * Executes command and broadcasts policy invalidation when policy is refreshed.
             * @param request TODO
             * @param context TODO
             * @return TODO
             */
            @Override
            public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
                if ("policy.refresh".equals(request.commandType())) {
                    String newVersion = request.payload() == null || request.payload().get("policyVersion") == null
                            ? context.policyVersion()
                            : String.valueOf(request.payload().get("policyVersion"));

                    Set<String> roles = context.roles() == null ? Set.of() : context.roles();
                    Set<String> denied = extractStringSet(request.payload(), "deniedCommandTypes");
                    PolicySnapshot snapshot = new PolicySnapshot(
                            context.tenantId(),
                            context.workspaceId(),
                            context.userId(),
                            newVersion,
                            roles,
                            denied,
                            Instant.now()
                    );
                    policySnapshotStore.upsert(snapshot);

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

            private Set<String> extractStringSet(Map<String, Object> payload, String key) {
                if (payload == null) {
                    return Set.of();
                }
                Object value = payload.get(key);
                if (!(value instanceof List<?> listValue)) {
                    return Set.of();
                }
                return listValue.stream()
                        .filter(item -> item != null && !String.valueOf(item).isBlank())
                        .map(String::valueOf)
                        .collect(java.util.stream.Collectors.toUnmodifiableSet());
            }
        };
    }

    /**
     * Provides default row-level data permission resolver for MyBatis integration.
     * @return TODO
     */
    @Bean
    public DataPermissionResolver dataPermissionResolver() {
        return new DefaultDataPermissionResolver();
    }

    /**
     * Composes command pipeline with authorization, egress, audit, and event publication.
     * @param authorizationGate TODO
     * @param egressGate TODO
     * @param handlers TODO
     * @param eventPublisher TODO
     * @param auditEventStore TODO
     * @return TODO
     */
    @Bean
    public CommandPipeline commandPipeline(
            AuthorizationGate authorizationGate,
            EgressGate egressGate,
            List<CommandHandler> handlers,
            DomainEventPublisher eventPublisher,
            AuditEventStore auditEventStore
    ) {
        return new CommandPipeline(authorizationGate, egressGate, handlers, eventPublisher, auditEventStore);
    }
}
