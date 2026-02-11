/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow template patch request contract.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
