/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Redis pubsub publisher for policy invalidation events.
 */
package com.ysmjjsy.goyais.capability.cache.policy;

import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationEvent;
import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationPublisher;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import org.springframework.data.redis.core.StringRedisTemplate;

/**
 * Uses Redis topic fan-out to invalidate policy caches across resource servers.
 */
public final class RedisPolicyInvalidationPublisher implements PolicyInvalidationPublisher {
    private final StringRedisTemplate redisTemplate;
    private final String channel;

    /**
     * Creates one publisher bound to a fixed Redis topic channel.
     */
    public RedisPolicyInvalidationPublisher(StringRedisTemplate redisTemplate, String channel) {
        this.redisTemplate = redisTemplate;
        this.channel = channel;
    }

    /**
     * Publishes one event payload to the configured Redis channel.
     */
    @Override
    public void publish(PolicyInvalidationEvent event) {
        redisTemplate.convertAndSend(channel, encode(event));
    }

    private String encode(PolicyInvalidationEvent event) {
        return String.join("|",
                encodePart(event.tenantId()),
                encodePart(event.workspaceId()),
                encodePart(event.userId()),
                encodePart(event.policyVersion()),
                encodePart(event.traceId()),
                encodePart(event.changedAt() == null ? "" : event.changedAt().toString())
        );
    }

    private String encodePart(String value) {
        return URLEncoder.encode(value == null ? "" : value, StandardCharsets.UTF_8);
    }
}
