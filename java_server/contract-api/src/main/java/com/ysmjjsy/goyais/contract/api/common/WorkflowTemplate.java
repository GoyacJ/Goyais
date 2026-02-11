/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow template contract model aligned with Go API semantics.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Represents one workflow template resource returned by workflow template APIs.
 */
public record WorkflowTemplate(
        ResourceBase base,
        String name,
        String description,
        Map<String, Object> graph,
        Map<String, Object> schemaInputs,
        Map<String, Object> schemaOutputs,
        Map<String, Object> uiState,
        int currentVersion
) {
}
