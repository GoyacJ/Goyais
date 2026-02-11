/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Distributed lock facade abstraction for critical command flow.
 */
package com.ysmjjsy.goyais.capability.cache;

import java.time.Duration;
import java.util.function.Supplier;

public interface LockFacade {
    <T> T withLock(String key, Duration waitTime, Duration leaseTime, Supplier<T> action);
}
