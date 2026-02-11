/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset lineage edge contract record.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;

/**
 * Describes one edge in an asset lineage graph.
 */
public record AssetLineageEdge(
        String id,
        String sourceAssetId,
        String targetAssetId,
        String runId,
        String stepId,
        String relation,
        Instant createdAt
) {
}
