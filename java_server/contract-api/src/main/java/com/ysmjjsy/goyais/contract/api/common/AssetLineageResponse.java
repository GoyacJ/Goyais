/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset lineage response envelope.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.util.List;

/**
 * Wraps lineage edges for one asset identifier.
 */
public record AssetLineageResponse(
        String assetId,
        List<AssetLineageEdge> edges
) {
}
