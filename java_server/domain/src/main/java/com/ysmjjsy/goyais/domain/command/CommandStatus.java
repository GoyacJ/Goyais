/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Canonical command lifecycle states.
 */
package com.ysmjjsy.goyais.domain.command;

public enum CommandStatus {
    ACCEPTED,
    RUNNING,
    SUCCEEDED,
    FAILED,
    CANCELED
}
