/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow run create request contract.
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
