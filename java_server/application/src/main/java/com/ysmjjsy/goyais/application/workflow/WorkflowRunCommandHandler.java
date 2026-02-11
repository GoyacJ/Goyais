/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command handler for workflow run create/cancel actions.
 */
package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Locale;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

/**
 * Executes workflow run commands while enforcing execute permissions.
 */
@Component
@Order(260)
public final class WorkflowRunCommandHandler implements CommandHandler {
    private final WorkflowTemplateRepository templateRepository;
    private final WorkflowRunRepository runRepository;
    private final boolean workflowEnabled;

    /**
     * Creates handler with workflow repositories and feature gate.
     */
    public WorkflowRunCommandHandler(
            WorkflowTemplateRepository templateRepository,
            WorkflowRunRepository runRepository,
            @Value("${goyais.feature.workflow-enabled:true}") boolean workflowEnabled
    ) {
        this.templateRepository = templateRepository;
        this.runRepository = runRepository;
        this.workflowEnabled = workflowEnabled;
    }

    /**
     * Returns true when command type belongs to workflow run actions.
     */
    @Override
    public boolean supports(String commandType) {
        return "workflow.run".equals(commandType) || "workflow.cancel".equals(commandType);
    }

    /**
     * Executes workflow run command and returns command result payload.
     */
    @Override
    public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
        ensureWorkflowEnabled();
        return switch (request.commandType()) {
            case "workflow.run" -> handleCreateRun(request.payload(), context);
            case "workflow.cancel" -> handleCancelRun(request.payload(), context);
            default -> throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        };
    }

    private Map<String, Object> handleCreateRun(Map<String, Object> payload, ExecutionContext context) {
        String templateId = requiredString(payload, "templateId");

        WorkflowTemplate template = templateRepository.findByIdInScope(templateId, context);
        if (template == null) {
            throw ContractException.of(404, "WORKFLOW_TEMPLATE_NOT_FOUND", "error.workflow.not_found");
        }
        ensureTemplateExecutePermission(template, context);

        if (!"published".equalsIgnoreCase(template.base().status())) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        String mode = parseMode(readString(payload, "mode"));
        String fromStepKey = readString(payload, "fromStepKey");
        boolean testNode = Boolean.TRUE.equals(payload == null ? null : payload.get("testNode"));
        Visibility visibility = parseVisibilityOrDefault(readString(payload, "visibility"), template.base().visibility());

        if (visibility == Visibility.TENANT || visibility == Visibility.PUBLIC) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "visibility_escalation_not_allowed")
            );
        }

        Map<String, Object> inputs = readObjectMap(payload, "inputs");
        WorkflowRun run = runRepository.createRun(
                context,
                template,
                visibility,
                mode,
                fromStepKey,
                testNode,
                inputs,
                Instant.now()
        );

        return Map.of("run", toRunPayload(run));
    }

    private Map<String, Object> handleCancelRun(Map<String, Object> payload, ExecutionContext context) {
        String runId = requiredString(payload, "runId");
        WorkflowRun run = runRepository.findByIdInScope(runId, context);
        if (run == null) {
            throw ContractException.of(404, "WORKFLOW_RUN_NOT_FOUND", "error.workflow.not_found");
        }
        ensureRunExecutePermission(run, context);

        WorkflowRun canceled = runRepository.cancelRun(runId, context, Instant.now());
        return Map.of("run", toRunPayload(canceled));
    }

    private void ensureWorkflowEnabled() {
        if (!workflowEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.workflow.not_implemented");
        }
    }

    private void ensureTemplateExecutePermission(WorkflowTemplate template, ExecutionContext context) {
        if (template.base().ownerId().equals(context.userId())) {
            return;
        }
        boolean allowed = templateRepository.hasPermission(template.base().id(), context, Permission.EXECUTE, Instant.now());
        if (!allowed) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "permission_denied")
            );
        }
    }

    private void ensureRunExecutePermission(WorkflowRun run, ExecutionContext context) {
        if (run.base().ownerId().equals(context.userId())) {
            return;
        }
        boolean allowed = runRepository.hasPermission(run.base().id(), context, Permission.EXECUTE, Instant.now());
        if (!allowed) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "permission_denied")
            );
        }
    }

    private String parseMode(String raw) {
        String mode = raw == null ? "" : raw.trim().toLowerCase(Locale.ROOT);
        return switch (mode) {
            case "", "sync" -> "sync";
            case "running" -> "running";
            case "fail" -> "fail";
            default -> throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        };
    }

    private Visibility parseVisibilityOrDefault(String raw, Visibility fallback) {
        if (raw == null || raw.isBlank()) {
            return fallback == null ? Visibility.PRIVATE : fallback;
        }
        try {
            return Visibility.valueOf(raw.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
    }

    private String requiredString(Map<String, Object> payload, String key) {
        String value = readString(payload, key);
        if (value.isBlank()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        return value;
    }

    private String readString(Map<String, Object> payload, String key) {
        if (payload == null || payload.get(key) == null) {
            return "";
        }
        return String.valueOf(payload.get(key)).trim();
    }

    private Map<String, Object> readObjectMap(Map<String, Object> payload, String key) {
        if (!(payload != null && payload.get(key) instanceof Map<?, ?> value)) {
            return Map.of();
        }
        Map<String, Object> copied = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : value.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            copied.put(String.valueOf(entry.getKey()), entry.getValue());
        }
        return Map.copyOf(copied);
    }

    private Map<String, Object> toRunPayload(WorkflowRun run) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("id", run.base().id());
        payload.put("tenantId", run.base().tenantId());
        payload.put("workspaceId", run.base().workspaceId());
        payload.put("ownerId", run.base().ownerId());
        payload.put("traceId", run.traceId());
        payload.put("visibility", run.base().visibility().name());
        payload.put("acl", run.base().acl());
        payload.put("status", run.base().status());
        payload.put("templateId", run.templateId());
        payload.put("templateVersion", run.templateVersion());
        payload.put("attempt", run.attempt());
        payload.put("retryOfRunId", run.retryOfRunId());
        payload.put("replayFromStepKey", run.replayFromStepKey());
        payload.put("inputs", run.inputs());
        payload.put("outputs", run.outputs());
        payload.put("startedAt", run.startedAt());
        payload.put("finishedAt", run.finishedAt());
        payload.put("durationMs", run.durationMs());
        payload.put("createdAt", run.base().createdAt());
        payload.put("updatedAt", run.base().updatedAt());
        if (run.error() != null) {
            payload.put("error", Map.of(
                    "code", run.error().code(),
                    "messageKey", run.error().messageKey()
            ));
        }
        return payload;
    }
}
