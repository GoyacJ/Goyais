/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>ACL permission set aligned with PRD and Go contract.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
