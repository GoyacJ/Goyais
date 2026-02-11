/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unified cache facade abstraction for spring cache/redisson strategy.
 */
package com.ysmjjsy.goyais.capability.cache;

import java.time.Duration;
import java.util.Optional;

public interface CacheFacade {
    Optional<String> get(String region, String key);

    void put(String region, String key, String value, Duration ttl);

    void evict(String region, String key);
}
