/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Publisher SPI for dynamic policy invalidation fan-out.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
