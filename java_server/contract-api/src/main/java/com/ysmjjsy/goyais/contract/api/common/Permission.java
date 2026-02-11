/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: ACL permission set aligned with PRD and Go contract.
 */
package com.ysmjjsy.goyais.contract.api.common;

/**
 * Defines ACL permissions supported by the cross-stack contract.
 */
public enum Permission {
    READ,
    WRITE,
    EXECUTE,
    MANAGE,
    SHARE
}
