/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Policy snapshot provider SPI for dynamic authorization and data permission.
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
