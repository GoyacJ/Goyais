/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset patch request contract for name/visibility/metadata updates.
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
