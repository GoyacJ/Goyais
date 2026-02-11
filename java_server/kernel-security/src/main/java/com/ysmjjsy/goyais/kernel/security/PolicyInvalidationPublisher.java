/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Publisher SPI for dynamic policy invalidation fan-out.
 */
package com.ysmjjsy.goyais.kernel.security;

/**
 * Broadcasts policy invalidation events to peer nodes.
 */
public interface PolicyInvalidationPublisher {

    /**
     * Publishes one invalidation event to the configured message channel.
     */
    void publish(PolicyInvalidationEvent event);
}
