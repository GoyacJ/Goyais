/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Workflow template REST controller aligned with Go-compatible /api/v1/workflow-templates routes.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.application.workflow.WorkflowTemplateApplicationService;
import com.ysmjjsy.goyais.contract.api.common.ErrorEnvelope;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplateCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplatePatchRequest;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.util.List;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * Exposes workflow template list/detail and domain-sugar write APIs.
 */
@RestController
@RequestMapping("/api/v1/workflow-templates")
public final class WorkflowTemplateController {
    private final WorkflowTemplateApplicationService templateService;
    private final RequestExecutionContextFactory executionContextFactory;
    private final boolean workflowEnabled;

    /**
     * Creates controller with template service and execution-context resolver.
     */
    public WorkflowTemplateController(
            WorkflowTemplateApplicationService templateService,
            RequestExecutionContextFactory executionContextFactory,
            @Value("${goyais.feature.workflow-enabled:true}") boolean workflowEnabled
    ) {
        this.templateService = templateService;
        this.executionContextFactory = executionContextFactory;
        this.workflowEnabled = workflowEnabled;
    }

    /**
     * Returns readable workflow template list response with pageInfo envelope.
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
        List<WorkflowTemplate> items = templateService.list(context, page, pageSize);
        long total = templateService.count(context);
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
     * Creates one workflow template through command-first domain sugar endpoint.
     */
    @PostMapping
    public ResponseEntity<WriteResponse<WorkflowTemplate>> create(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestBody WorkflowTemplateCreateRequest request
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(templateService.create(request, context));
    }

    /**
     * Returns one readable workflow template or not-found contract envelope.
     */
    @GetMapping("/{templateId}")
    public ResponseEntity<?> get(
            @PathVariable String templateId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        WorkflowTemplate template = templateService.get(templateId, context);
        if (template == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of(
                    "WORKFLOW_TEMPLATE_NOT_FOUND",
                    "error.workflow.not_found"
            ));
        }
        return ResponseEntity.ok(template);
    }

    /**
     * Patches one workflow template through command-first domain sugar endpoint.
     */
    @PostMapping("/{templateId}:patch")
    public ResponseEntity<WriteResponse<WorkflowTemplate>> patch(
            @PathVariable String templateId,
            @RequestBody WorkflowTemplatePatchRequest request,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(templateService.patch(templateId, request, context));
    }

    /**
     * Publishes one workflow template through command-first domain sugar endpoint.
     */
    @PostMapping("/{templateId}:publish")
    public ResponseEntity<WriteResponse<WorkflowTemplate>> publish(
            @PathVariable String templateId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureWorkflowEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(templateService.publish(templateId, context));
    }

    private void ensureWorkflowEnabled() {
        if (!workflowEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.workflow.not_implemented");
        }
    }
}
