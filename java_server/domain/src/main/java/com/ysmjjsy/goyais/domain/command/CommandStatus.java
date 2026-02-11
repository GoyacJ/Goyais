/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Canonical command lifecycle states.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.domain.command;

/**
 * Defines canonical command lifecycle states used by audit and read models.
 */
public enum CommandStatus {
    ACCEPTED,
    RUNNING,
    SUCCEEDED,
    FAILED,
    CANCELED
}
