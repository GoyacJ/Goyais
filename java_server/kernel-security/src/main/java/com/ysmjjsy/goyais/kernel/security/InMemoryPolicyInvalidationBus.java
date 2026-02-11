/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: In-process invalidation bus fallback when Redis pubsub is not configured.
 */
package com.ysmjjsy.goyais.kernel.security;

import java.util.List;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.function.Consumer;

/**
 * Provides local fan-out semantics for policy invalidation in minimal mode.
 */
public final class InMemoryPolicyInvalidationBus implements PolicyInvalidationPublisher, PolicyInvalidationSubscriber {
    private final List<Consumer<PolicyInvalidationEvent>> listeners = new CopyOnWriteArrayList<>();

    /**
     * Dispatches invalidation events to all local subscribers.
     */
    @Override
    public void publish(PolicyInvalidationEvent event) {
        for (Consumer<PolicyInvalidationEvent> listener : listeners) {
            listener.accept(event);
        }
    }

    /**
     * Registers one callback for future invalidation events.
     */
    @Override
    public void start(Consumer<PolicyInvalidationEvent> callback) {
        listeners.add(callback);
    }
}
