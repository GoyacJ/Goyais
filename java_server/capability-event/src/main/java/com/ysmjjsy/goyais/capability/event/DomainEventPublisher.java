/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Event publisher SPI used by command pipeline and outbox integration.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.event;

/**
 * Publishes domain events to outbox or message bus infrastructure.
 */
public interface DomainEventPublisher {

    /**
     * Publishes one domain event after command execution.
     */
    void publish(DomainEvent event);
}
