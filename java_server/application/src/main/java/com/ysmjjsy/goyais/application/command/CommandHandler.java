/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Command handler SPI for extensible command execution.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
