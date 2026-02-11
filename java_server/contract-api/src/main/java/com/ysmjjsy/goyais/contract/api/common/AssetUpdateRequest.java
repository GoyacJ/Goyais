/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset patch request contract for name/visibility/metadata updates.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Represents mutable fields accepted by the asset patch API.
 */
public record AssetUpdateRequest(
        String name,
        Visibility visibility,
        Map<String, Object> metadata
) {
}
