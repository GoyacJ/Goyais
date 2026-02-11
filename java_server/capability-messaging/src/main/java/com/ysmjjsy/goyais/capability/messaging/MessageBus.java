/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Message bus SPI with memory and kafka pluggable providers.
 */
package com.ysmjjsy.goyais.capability.messaging;

import com.ysmjjsy.goyais.capability.event.DomainEvent;
import java.util.function.Consumer;

/**
 * Defines pluggable event bus contract for memory and Kafka providers.
 */
public interface MessageBus {

    /**
     * Publishes one domain event to a topic.
     */
    void publish(String topic, DomainEvent event);

    /**
     * Subscribes one consumer to topic events.
     */
    void subscribe(String topic, Consumer<DomainEvent> consumer);
}
