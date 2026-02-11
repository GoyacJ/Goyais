/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Policy snapshot provider SPI for dynamic authorization and data permission.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

import com.ysmjjsy.goyais.kernel.core.ExecutionContext;

/**
 * Loads and invalidates policy snapshots for a user scope.
 */
public interface PolicySnapshotProvider {

    /**
     * Returns the latest policy snapshot visible to current request context.
     */
    PolicySnapshot loadLatest(ExecutionContext context);

    /**
     * Evicts one cached policy snapshot by tenant/workspace/user scope.
     */
    void evict(String tenantId, String workspaceId, String userId);
}
