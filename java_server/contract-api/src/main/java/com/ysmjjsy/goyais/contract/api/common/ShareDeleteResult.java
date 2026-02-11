/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Share delete result contract model.
 */
package com.ysmjjsy.goyais.contract.api.common;

/**
 * Represents delete confirmation returned by share delete endpoint.
 */
public record ShareDeleteResult(
        String id,
        String status
) {
}
