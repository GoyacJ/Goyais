/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command handler SPI for extensible command execution.
 */
package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.util.Map;

/**
 * Encapsulates one command execution strategy selected by commandType.
 */
public interface CommandHandler {

    /**
     * Returns true when this handler is responsible for the command type.
     */
    boolean supports(String commandType);

    /**
     * Executes command business logic in the caller execution context.
     */
    Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context);
}
