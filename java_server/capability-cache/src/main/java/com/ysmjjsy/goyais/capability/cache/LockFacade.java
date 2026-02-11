/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Distributed lock facade abstraction for critical command flow.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.cache;

import java.time.Duration;
import java.util.function.Supplier;

/**
 * Abstracts distributed lock execution semantics for critical sections.
 */
public interface LockFacade {

    /**
     * Executes action within lock boundaries and returns action result.
     */
    <T> T withLock(String key, Duration waitTime, Duration leaseTime, Supplier<T> action);
}
