/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Visibility levels shared by all resources.
 */
package com.ysmjjsy.goyais.contract.api.common;

/**
 * Represents supported visibility scopes for all shared resources.
 */
public enum Visibility {
    PRIVATE,
    WORKSPACE,
    TENANT,
    PUBLIC
}
