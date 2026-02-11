/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset lineage edge contract record.
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
