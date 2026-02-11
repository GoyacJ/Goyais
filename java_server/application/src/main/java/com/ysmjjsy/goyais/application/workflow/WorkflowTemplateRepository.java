/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Repository SPI for workflow template persistence and permission checks.
 */
package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;
import java.util.Map;

/**
 * Persists workflow template resources and evaluates template ACL permissions.
 */
public interface WorkflowTemplateRepository {

    /**
     * Creates one draft template and returns the persisted template resource.
     */
    WorkflowTemplate createDraft(
            ExecutionContext context,
            String name,
            String description,
            Visibility visibility,
            Map<String, Object> graph,
            Map<String, Object> schemaInputs,
            Map<String, Object> schemaOutputs,
            Map<String, Object> uiState,
            Instant now
    );

    /**
     * Returns one template in tenant/workspace scope without ACL read filtering.
     */
    WorkflowTemplate findByIdInScope(String templateId, ExecutionContext context);

    /**
     * Returns one readable template or null when inaccessible.
     */
    WorkflowTemplate findReadableById(String templateId, ExecutionContext context);

    /**
     * Returns readable templates ordered by created_at DESC and id DESC.
     */
    List<WorkflowTemplate> listReadable(ExecutionContext context, int page, int pageSize);

    /**
     * Returns count of readable templates for current context.
     */
    long countReadable(ExecutionContext context);

    /**
     * Updates template graph and uiState and returns latest resource.
     */
    WorkflowTemplate patch(
            String templateId,
            ExecutionContext context,
            Map<String, Object> graph,
            Map<String, Object> uiState,
            Instant now
    );

    /**
     * Publishes one template and returns latest resource.
     */
    WorkflowTemplate publish(String templateId, ExecutionContext context, Instant now);

    /**
     * Returns true when user or roles have requested template permission.
     */
    boolean hasPermission(String templateId, ExecutionContext context, Permission permission, Instant now);
}
