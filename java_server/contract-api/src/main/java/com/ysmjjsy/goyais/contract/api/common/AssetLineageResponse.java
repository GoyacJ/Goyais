/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset lineage response envelope.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
