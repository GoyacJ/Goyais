/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow template create request contract.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Carries fields required by workflow template create endpoint.
 */
public record WorkflowTemplateCreateRequest(
        String name,
        String description,
        Map<String, Object> graph,
        Map<String, Object> schemaInputs,
        Map<String, Object> schemaOutputs,
        Map<String, Object> uiState,
        Visibility visibility
) {
}
