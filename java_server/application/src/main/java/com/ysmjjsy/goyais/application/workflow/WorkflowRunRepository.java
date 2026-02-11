/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Repository SPI for workflow run, step, and event persistence.
 */
package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;
import java.util.Map;

/**
 * Persists workflow run resources and exposes step/event read models.
 */
public interface WorkflowRunRepository {

    /**
     * Creates one workflow run with initial step and events and returns persisted run.
     */
    WorkflowRun createRun(
            ExecutionContext context,
            WorkflowTemplate template,
            Visibility visibility,
            String mode,
            String fromStepKey,
            boolean testNode,
            Map<String, Object> inputs,
            Instant now
    );

    /**
     * Returns one run in tenant/workspace scope without ACL read filtering.
     */
    WorkflowRun findByIdInScope(String runId, ExecutionContext context);

    /**
     * Returns one readable run or null when inaccessible.
     */
    WorkflowRun findReadableById(String runId, ExecutionContext context);

    /**
     * Returns readable runs ordered by created_at DESC and id DESC.
     */
    List<WorkflowRun> listReadable(ExecutionContext context, int page, int pageSize);

    /**
     * Returns count of readable runs for current context.
     */
    long countReadable(ExecutionContext context);

    /**
     * Returns true when user or roles have requested run permission.
     */
    boolean hasPermission(String runId, ExecutionContext context, Permission permission, Instant now);

    /**
     * Cancels one run and pending/running steps and returns latest run resource.
     */
    WorkflowRun cancelRun(String runId, ExecutionContext context, Instant now);

    /**
     * Returns step runs for one run ordered by created_at DESC and id DESC.
     */
    List<StepRun> listSteps(String runId, ExecutionContext context, int page, int pageSize);

    /**
     * Returns count of step runs for one run in scope.
     */
    long countSteps(String runId, ExecutionContext context);

    /**
     * Returns run events for one run ordered by created_at ASC and id ASC.
     */
    List<WorkflowRunEvent> listEvents(String runId, ExecutionContext context);
}
