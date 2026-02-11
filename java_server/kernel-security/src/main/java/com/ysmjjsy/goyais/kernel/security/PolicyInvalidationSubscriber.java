/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Subscriber SPI for policy invalidation broadcast channels.
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
