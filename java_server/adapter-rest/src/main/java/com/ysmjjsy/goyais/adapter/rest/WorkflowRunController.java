/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow run REST controller aligned with Go-compatible /api/v1/workflow-runs routes.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.application.workflow.WorkflowRunApplicationService;
import com.ysmjjsy.goyais.contract.api.common.ErrorEnvelope;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.io.IOException;
import java.util.List;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

/**
 * Exposes workflow run list/detail, step list, and event stream APIs.
 */
@RestController
@RequestMapping("/api/v1/workflow-runs")
public final class WorkflowRunController {
    private final WorkflowRunApplicationService runService;
    private final RequestExecutionContextFactory executionContextFactory;
    private final boolean workflowEnabled;

    /**
     * Creates controller with workflow run service and execution-context resolver.
     */
    public WorkflowRunController(
            WorkflowRunApplicationService runService,
            RequestExecutionContextFactory executionContextFactory,
            @Value("${goyais.feature.workflow-enabled:true}") boolean workflowEnabled
    ) {
        this.runService = runService;
        this.executionContextFactory = executionContextFactory;
        this.workflowEnabled = workflowEnabled;
    }

    /**
     * Returns readable workflow run list response with pageInfo envelope.
     */
    @GetMapping
    public Map<String, Object> list(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestParam(value = "page", defaultValue = "1") int page,
            @RequestParam(value = "pageSize", defaultValue = "20") int pageSize
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        List<WorkflowRun> items = runService.list(context, page, pageSize);
        long total = runService.count(context);
        return Map.of(
                "items", items,
                "pageInfo", Map.of(
                        "page", Math.max(page, 1),
                        "pageSize", Math.min(Math.max(pageSize, 1), 200),
                        "total", total
                )
        );
    }

    /**
     * Creates one workflow run through command-first domain sugar endpoint.
     */
    @PostMapping
    public ResponseEntity<WriteResponse<WorkflowRun>> create(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestBody WorkflowRunCreateRequest request
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(runService.create(request, context));
    }

    /**
     * Returns one readable workflow run or not-found contract envelope.
     */
    @GetMapping("/{runId}")
    public ResponseEntity<?> get(
            @PathVariable String runId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        WorkflowRun run = runService.get(runId, context);
        if (run == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of(
                    "WORKFLOW_RUN_NOT_FOUND",
                    "error.workflow.not_found"
            ));
        }
        return ResponseEntity.ok(run);
    }

    /**
     * Cancels one workflow run through command-first domain sugar endpoint.
     */
    @PostMapping("/{runId}:cancel")
    public ResponseEntity<WriteResponse<WorkflowRun>> cancel(
            @PathVariable String runId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(runService.cancel(runId, context));
    }

    /**
     * Returns step run list for one readable workflow run.
     */
    @GetMapping("/{runId}/steps")
    public ResponseEntity<?> listSteps(
            @PathVariable String runId,
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestParam(value = "page", defaultValue = "1") int page,
            @RequestParam(value = "pageSize", defaultValue = "20") int pageSize
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        WorkflowRun run = runService.get(runId, context);
        if (run == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of(
                    "WORKFLOW_RUN_NOT_FOUND",
                    "error.workflow.not_found"
            ));
        }

        List<StepRun> items = runService.listSteps(runId, context, page, pageSize);
        long total = runService.countSteps(runId, context);
        return ResponseEntity.ok(Map.of(
                "items", items,
                "pageInfo", Map.of(
                        "page", Math.max(page, 1),
                        "pageSize", Math.min(Math.max(pageSize, 1), 200),
                        "total", total
                )
        ));
    }

    /**
     * Streams workflow run events as SSE for one readable workflow run.
     */
    @GetMapping(value = "/{runId}/events", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public ResponseEntity<?> streamEvents(
            @PathVariable String runId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        WorkflowRun run = runService.get(runId, context);
        if (run == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of(
                    "WORKFLOW_RUN_NOT_FOUND",
                    "error.workflow.not_found"
            ));
        }

        List<WorkflowRunEvent> events = runService.listEvents(runId, context);
        SseEmitter emitter = new SseEmitter(30_000L);
        try {
            for (WorkflowRunEvent event : events) {
                emitter.send(SseEmitter.event()
                        .id(event.id())
                        .name(event.eventType())
                        .data(event, MediaType.APPLICATION_JSON));
            }
            emitter.complete();
        } catch (IOException ex) {
            emitter.completeWithError(ex);
        }
        return ResponseEntity.ok(emitter);
    }

    private void ensureWorkflowEnabled() {
        if (!workflowEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.workflow.not_implemented");
        }
    }
}
