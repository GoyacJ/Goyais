/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Subscriber SPI for policy invalidation broadcast channels.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

import java.util.function.Consumer;

/**
 * Subscribes to invalidation events emitted by peer nodes.
 */
public interface PolicyInvalidationSubscriber {

    /**
     * Starts subscription and forwards incoming events to the callback.
     */
    void start(Consumer<PolicyInvalidationEvent> callback);
}
