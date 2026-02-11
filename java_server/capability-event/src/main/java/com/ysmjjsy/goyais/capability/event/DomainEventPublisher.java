/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Event publisher SPI used by command pipeline and outbox integration.
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
