/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Durable store SPI for policy snapshots used by dynamic authorization.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.kernel.security;

/**
 * Provides durable read/write access for policy snapshots across node restarts.
 */
public interface PolicySnapshotStore {

    /**
     * Loads the latest policy snapshot for one tenant/workspace/user scope.
     */
    PolicySnapshot load(String tenantId, String workspaceId, String userId);

    /**
     * Persists or updates one policy snapshot in the durable store.
     */
    void upsert(PolicySnapshot snapshot);
}
