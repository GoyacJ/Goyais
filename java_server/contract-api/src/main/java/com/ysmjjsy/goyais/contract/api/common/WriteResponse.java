/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Generic write response wrapper for resource and command ref.
 */
package com.ysmjjsy.goyais.contract.api.common;

public record WriteResponse<T>(
        T resource,
        CommandRef commandRef
) {
}
