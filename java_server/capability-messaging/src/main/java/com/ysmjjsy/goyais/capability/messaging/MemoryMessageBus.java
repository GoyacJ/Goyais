/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>In-memory message bus fallback for minimal runtime mode.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.messaging;

import com.ysmjjsy.goyais.capability.event.DomainEvent;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.function.Consumer;

public final class MemoryMessageBus implements MessageBus {
    private final Map<String, List<Consumer<DomainEvent>>> subscribers = new ConcurrentHashMap<>();

    /**
     * Delivers the event to all subscribers currently registered on the topic.
     * @param topic TODO
     * @param event TODO
     */
    @Override
    public void publish(String topic, DomainEvent event) {
        subscribers.getOrDefault(topic, List.of()).forEach(consumer -> consumer.accept(event));
    }

    /**
     * Registers one subscriber callback for subsequent topic events.
     * @param topic TODO
     * @param consumer TODO
     */
    @Override
    public void subscribe(String topic, Consumer<DomainEvent> consumer) {
        subscribers.computeIfAbsent(topic, ignored -> new CopyOnWriteArrayList<>()).add(consumer);
    }
}
