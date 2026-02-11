/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Redis-first policy snapshot provider with durable-store fallback.
 */
package com.ysmjjsy.goyais.capability.cache.policy;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshot;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshotProvider;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshotStore;
import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import org.springframework.data.redis.core.StringRedisTemplate;

/**
 * Resolves policy snapshots from Redis cache and falls back to durable store on misses.
 */
public final class RedisPolicySnapshotProvider implements PolicySnapshotProvider {
    private static final String CACHE_PREFIX = "goyais:policy:snapshot:";

    private final StringRedisTemplate redisTemplate;
    private final PolicySnapshotStore snapshotStore;
    private final Duration cacheTtl;
    private final ObjectMapper objectMapper;
    private final ConcurrentMap<String, CacheEntry> localCache = new ConcurrentHashMap<>();

    /**
     * Creates one provider using Redis cache, durable store, and in-process fallback cache.
     */
    public RedisPolicySnapshotProvider(
            StringRedisTemplate redisTemplate,
            PolicySnapshotStore snapshotStore,
            Duration cacheTtl,
            ObjectMapper objectMapper
    ) {
        this.redisTemplate = redisTemplate;
        this.snapshotStore = snapshotStore;
        this.cacheTtl = cacheTtl;
        this.objectMapper = objectMapper;
    }

    /**
     * Loads latest snapshot from local cache, Redis, durable store, or request defaults.
     */
    @Override
    public PolicySnapshot loadLatest(ExecutionContext context) {
        String scopeKey = scopeKey(context.tenantId(), context.workspaceId(), context.userId());
        PolicySnapshot local = loadFromLocal(scopeKey);
        if (local != null) {
            return local;
        }

        PolicySnapshot cached;
        try {
            cached = loadFromRedis(scopeKey);
        } catch (RuntimeException ex) {
            cached = null;
        }
        if (cached != null) {
            putLocal(scopeKey, cached);
            return cached;
        }

        PolicySnapshot stored = snapshotStore.load(context.tenantId(), context.workspaceId(), context.userId());
        PolicySnapshot effective = stored == null ? fallbackFromContext(context) : merge(stored, context);

        safeWriteRedis(scopeKey, effective);
        putLocal(scopeKey, effective);
        return effective;
    }

    /**
     * Evicts local and Redis cache entries for one policy scope.
     */
    @Override
    public void evict(String tenantId, String workspaceId, String userId) {
        String scopeKey = scopeKey(tenantId, workspaceId, userId);
        localCache.remove(scopeKey);
        try {
            redisTemplate.delete(redisKey(scopeKey));
        } catch (RuntimeException ex) {
            // Redis-unavailable path intentionally falls back to local cache only.
        }
    }

    private PolicySnapshot loadFromLocal(String scopeKey) {
        CacheEntry entry = localCache.get(scopeKey);
        if (entry == null) {
            return null;
        }
        if (entry.expiresAt().isBefore(Instant.now())) {
            localCache.remove(scopeKey);
            return null;
        }
        return entry.snapshot();
    }

    private PolicySnapshot loadFromRedis(String scopeKey) {
        String payload = redisTemplate.opsForValue().get(redisKey(scopeKey));
        if (payload == null || payload.isBlank()) {
            return null;
        }
        try {
            SnapshotPayload decoded = objectMapper.readValue(payload, SnapshotPayload.class);
            return new PolicySnapshot(
                    decoded.tenantId,
                    decoded.workspaceId,
                    decoded.userId,
                    decoded.policyVersion,
                    decoded.roles == null ? Set.of() : Set.copyOf(decoded.roles),
                    decoded.deniedCommandTypes == null ? Set.of() : Set.copyOf(decoded.deniedCommandTypes),
                    decoded.updatedAt == null ? Instant.now() : decoded.updatedAt
            );
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to decode policy snapshot cache", ex);
        }
    }

    private void safeWriteRedis(String scopeKey, PolicySnapshot snapshot) {
        try {
            String payload = objectMapper.writeValueAsString(new SnapshotPayload(snapshot));
            redisTemplate.opsForValue().set(redisKey(scopeKey), payload, cacheTtl);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to encode policy snapshot cache", ex);
        } catch (RuntimeException ex) {
            // Redis-unavailable path intentionally falls back to local cache only.
        }
    }

    private void putLocal(String scopeKey, PolicySnapshot snapshot) {
        localCache.put(scopeKey, new CacheEntry(snapshot, Instant.now().plus(cacheTtl)));
    }

    private PolicySnapshot fallbackFromContext(ExecutionContext context) {
        return new PolicySnapshot(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                normalizeVersion(context.policyVersion()),
                context.roles() == null ? Set.of() : context.roles(),
                Set.of(),
                Instant.now()
        );
    }

    private PolicySnapshot merge(PolicySnapshot stored, ExecutionContext context) {
        Set<String> mergedRoles = new java.util.HashSet<>();
        if (stored.roles() != null) {
            mergedRoles.addAll(stored.roles());
        }
        if (context.roles() != null) {
            mergedRoles.addAll(context.roles());
        }

        String requestVersion = normalizeVersion(context.policyVersion());
        String effectiveVersion = normalizeVersion(stored.policyVersion());
        String selectedVersion = requestVersion.compareTo(effectiveVersion) > 0 ? requestVersion : effectiveVersion;

        return new PolicySnapshot(
                stored.tenantId(),
                stored.workspaceId(),
                stored.userId(),
                selectedVersion,
                Set.copyOf(mergedRoles),
                stored.deniedCommandTypes() == null ? Set.of() : stored.deniedCommandTypes(),
                Instant.now()
        );
    }

    private String normalizeVersion(String value) {
        return value == null || value.isBlank() ? "v0.1" : value.trim();
    }

    private String redisKey(String scopeKey) {
        return CACHE_PREFIX + scopeKey;
    }

    private String scopeKey(String tenantId, String workspaceId, String userId) {
        return String.join("::",
                tenantId == null ? "" : tenantId,
                workspaceId == null ? "" : workspaceId,
                userId == null ? "" : userId
        );
    }

    private record CacheEntry(PolicySnapshot snapshot, Instant expiresAt) {
    }

    private static final class SnapshotPayload {
        public String tenantId;
        public String workspaceId;
        public String userId;
        public String policyVersion;
        public Set<String> roles;
        public Set<String> deniedCommandTypes;
        public Instant updatedAt;

        /**
         * Creates an empty payload instance for Jackson decoding.
         */
        public SnapshotPayload() {
        }

        /**
         * Creates one payload snapshot from domain model.
         */
        SnapshotPayload(PolicySnapshot snapshot) {
            this.tenantId = snapshot.tenantId();
            this.workspaceId = snapshot.workspaceId();
            this.userId = snapshot.userId();
            this.policyVersion = snapshot.policyVersion();
            this.roles = snapshot.roles();
            this.deniedCommandTypes = snapshot.deniedCommandTypes();
            this.updatedAt = snapshot.updatedAt();
        }
    }
}
