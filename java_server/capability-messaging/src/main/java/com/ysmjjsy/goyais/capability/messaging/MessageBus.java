/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Message bus SPI with memory and kafka pluggable providers.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
