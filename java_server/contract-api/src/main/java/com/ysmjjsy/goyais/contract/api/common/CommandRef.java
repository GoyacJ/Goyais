/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command reference returned by domain sugar write APIs.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;

/**
 * References accepted command metadata returned by write APIs.
 */
public record CommandRef(
        String commandId,
        String status,
        Instant acceptedAt
) {
}
