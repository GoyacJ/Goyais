/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow template create request contract.
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
