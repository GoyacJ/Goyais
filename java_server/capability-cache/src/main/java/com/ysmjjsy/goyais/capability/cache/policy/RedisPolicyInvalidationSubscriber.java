/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Redis pubsub subscriber for policy invalidation events.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.cache.policy;

import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationEvent;
import com.ysmjjsy.goyais.kernel.security.PolicyInvalidationSubscriber;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.function.Consumer;
import org.springframework.data.redis.connection.Message;
import org.springframework.data.redis.connection.MessageListener;
import org.springframework.data.redis.listener.ChannelTopic;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;

/**
 * Receives invalidation payloads from Redis and forwards them to local consumers.
 */
public final class RedisPolicyInvalidationSubscriber implements PolicyInvalidationSubscriber {
    private final RedisMessageListenerContainer listenerContainer;
    private final String channel;

    /**
     * Creates one subscriber bound to a fixed Redis topic channel.
     * @param listenerContainer TODO
     * @param channel TODO
     */
    public RedisPolicyInvalidationSubscriber(
            RedisMessageListenerContainer listenerContainer,
            String channel
    ) {
        this.listenerContainer = listenerContainer;
        this.channel = channel;
    }

    /**
     * Starts asynchronous event forwarding from Redis pubsub to callback consumer.
     * @param callback TODO
     */
    @Override
    public void start(Consumer<PolicyInvalidationEvent> callback) {
        listenerContainer.addMessageListener(new MessageListener() {
            /**
             * Parses Redis payload and forwards valid events to the callback.
             * @param message TODO
             * @param pattern TODO
             */
            @Override
            public void onMessage(Message message, byte[] pattern) {
                String payload = new String(message.getBody(), StandardCharsets.UTF_8);
                try {
                    PolicyInvalidationEvent event = decode(payload);
                    callback.accept(event);
                } catch (Exception ex) {
                    throw new IllegalStateException("failed to parse policy invalidation payload", ex);
                }
            }
        }, new ChannelTopic(channel));
    }

    private PolicyInvalidationEvent decode(String payload) {
        String[] parts = payload.split("\\|", -1);
        if (parts.length != 6) {
            throw new IllegalArgumentException("invalid invalidation payload");
        }
        String changedAt = decodePart(parts[5]);
        return new PolicyInvalidationEvent(
                decodePart(parts[0]),
                decodePart(parts[1]),
                decodePart(parts[2]),
                decodePart(parts[3]),
                decodePart(parts[4]),
                changedAt.isBlank() ? null : Instant.parse(changedAt)
        );
    }

    private String decodePart(String value) {
        return URLDecoder.decode(value, StandardCharsets.UTF_8);
    }
}
