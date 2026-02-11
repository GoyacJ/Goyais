/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow template patch request contract.
 */
package com.ysmjjsy.goyais.contract.api.common;

import java.util.List;
import java.util.Map;

/**
 * Carries graph-level patch fields accepted by workflow template patch endpoint.
 */
public record WorkflowTemplatePatchRequest(
        Map<String, Object> graph,
        List<Map<String, Object>> operations
) {
}
