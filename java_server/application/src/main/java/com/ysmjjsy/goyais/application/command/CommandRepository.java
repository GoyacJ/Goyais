/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Repository SPI for persisted command read/write operations.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.util.List;

/**
 * Persists and queries command resources with data-permission-aware filtering.
 */
public interface CommandRepository {

    /**
     * Persists one command resource.
     */
    void save(CommandResource resource);

    /**
     * Returns readable commands for the current caller with stable ordering.
     */
    List<CommandResource> listReadable(ExecutionContext context, int limit);

    /**
     * Returns one readable command by identifier, or null when inaccessible.
     */
    CommandResource findReadableById(String commandId, ExecutionContext context);
}
