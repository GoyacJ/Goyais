/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unified cache facade abstraction for spring cache/redisson strategy.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.cache;

import java.time.Duration;
import java.util.Optional;

/**
 * Abstracts cache read/write/evict operations across cache providers.
 */
public interface CacheFacade {

    /**
     * Reads one cache value by region and key.
     */
    Optional<String> get(String region, String key);

    /**
     * Writes one value with explicit TTL.
     */
    void put(String region, String key, String value, Duration ttl);

    /**
     * Removes one cache entry from the configured provider.
     */
    void evict(String region, String key);
}
