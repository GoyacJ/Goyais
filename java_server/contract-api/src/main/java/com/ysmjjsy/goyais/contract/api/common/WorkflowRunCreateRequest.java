/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow run create request contract.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.util.Map;

/**
 * Carries fields required to create one workflow run.
 */
public record WorkflowRunCreateRequest(
        String templateId,
        String templateVersion,
        Map<String, Object> inputs,
        String mode,
        String fromStepKey,
        Boolean testNode,
        Visibility visibility
) {
}
