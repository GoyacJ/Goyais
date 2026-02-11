/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command handler for workflow template create/patch/publish actions.
 */
package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

/**
 * Executes workflow template commands while enforcing owner/ACL authorization.
 */
@Component
@Order(240)
public final class WorkflowTemplateCommandHandler implements CommandHandler {
    private final WorkflowTemplateRepository templateRepository;
    private final boolean workflowEnabled;

    /**
     * Creates handler with template repository and workflow feature gate.
     */
    public WorkflowTemplateCommandHandler(
            WorkflowTemplateRepository templateRepository,
            @Value("${goyais.feature.workflow-enabled:true}") boolean workflowEnabled
    ) {
        this.templateRepository = templateRepository;
        this.workflowEnabled = workflowEnabled;
    }

    /**
     * Returns true when command type belongs to workflow template actions.
     */
    @Override
    public boolean supports(String commandType) {
        return "workflow.createDraft".equals(commandType)
                || "workflow.patch".equals(commandType)
                || "workflow.publish".equals(commandType);
    }

    /**
     * Executes workflow template command and returns command result payload.
     */
    @Override
    public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
        ensureWorkflowEnabled();
        return switch (request.commandType()) {
            case "workflow.createDraft" -> handleCreateDraft(request.payload(), context);
            case "workflow.patch" -> handlePatch(request.payload(), context);
            case "workflow.publish" -> handlePublish(request.payload(), context);
            default -> throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        };
    }

    private Map<String, Object> handleCreateDraft(Map<String, Object> payload, ExecutionContext context) {
        String name = requiredString(payload, "name");
        if (!(payload != null && payload.get("graph") instanceof Map<?, ?> graphRaw)) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Visibility visibility = parseVisibility(readString(payload, "visibility"));
        if (visibility == Visibility.TENANT || visibility == Visibility.PUBLIC) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "visibility_escalation_not_allowed")
            );
        }

        WorkflowTemplate created = templateRepository.createDraft(
                context,
                name,
                readString(payload, "description"),
                visibility,
                copyObjectMap(graphRaw),
                readObjectMap(payload, "schemaInputs"),
                readObjectMap(payload, "schemaOutputs"),
                readObjectMap(payload, "uiState"),
                Instant.now()
        );
        return Map.of("template", toTemplatePayload(created));
    }

    private Map<String, Object> handlePatch(Map<String, Object> payload, ExecutionContext context) {
        String templateId = requiredString(payload, "templateId");
        if (!(payload != null && payload.get("patch") instanceof Map<?, ?> patchRaw)) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        Map<String, Object> patch = copyObjectMap(patchRaw);

        WorkflowTemplate existing = templateRepository.findByIdInScope(templateId, context);
        if (existing == null) {
            throw ContractException.of(404, "WORKFLOW_TEMPLATE_NOT_FOUND", "error.workflow.not_found");
        }
        ensureTemplatePermission(existing, context, Permission.WRITE);

        if ("disabled".equalsIgnoreCase(existing.base().status())) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> nextGraph;
        if (patch.get("graph") instanceof Map<?, ?> graphRaw) {
            nextGraph = copyObjectMap(graphRaw);
        } else if (patch.get("operations") instanceof List<?> operationsRaw) {
            List<Map<String, Object>> operations = toObjectOperationList(operationsRaw);
            nextGraph = WorkflowPatchApplier.apply(existing.graph(), operations);
        } else {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        WorkflowTemplate updated = templateRepository.patch(
                templateId,
                context,
                nextGraph,
                Map.of("lastPatch", patch),
                Instant.now()
        );
        return Map.of("template", toTemplatePayload(updated));
    }

    private Map<String, Object> handlePublish(Map<String, Object> payload, ExecutionContext context) {
        String templateId = requiredString(payload, "templateId");

        WorkflowTemplate existing = templateRepository.findByIdInScope(templateId, context);
        if (existing == null) {
            throw ContractException.of(404, "WORKFLOW_TEMPLATE_NOT_FOUND", "error.workflow.not_found");
        }
        ensureTemplatePermission(existing, context, Permission.MANAGE);

        if ("disabled".equalsIgnoreCase(existing.base().status())) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        WorkflowTemplate published = templateRepository.publish(templateId, context, Instant.now());
        return Map.of("template", toTemplatePayload(published));
    }

    private void ensureWorkflowEnabled() {
        if (!workflowEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.workflow.not_implemented");
        }
    }

    private void ensureTemplatePermission(WorkflowTemplate template, ExecutionContext context, Permission permission) {
        if (template.base().ownerId().equals(context.userId())) {
            return;
        }
        boolean allowed = templateRepository.hasPermission(template.base().id(), context, permission, Instant.now());
        if (!allowed) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "permission_denied")
            );
        }
    }

    private Visibility parseVisibility(String raw) {
        if (raw == null || raw.isBlank()) {
            return Visibility.PRIVATE;
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
        return copyObjectMap(value);
    }

    private List<Map<String, Object>> toObjectOperationList(List<?> raw) {
        java.util.ArrayList<Map<String, Object>> operations = new java.util.ArrayList<>();
        for (Object item : raw) {
            if (item instanceof Map<?, ?> mapItem) {
                operations.add(copyObjectMap(mapItem));
            }
        }
        if (operations.isEmpty()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        return operations;
    }

    private Map<String, Object> toTemplatePayload(WorkflowTemplate template) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("id", template.base().id());
        payload.put("tenantId", template.base().tenantId());
        payload.put("workspaceId", template.base().workspaceId());
        payload.put("ownerId", template.base().ownerId());
        payload.put("visibility", template.base().visibility().name());
        payload.put("acl", template.base().acl());
        payload.put("status", template.base().status());
        payload.put("name", template.name());
        payload.put("description", template.description());
        payload.put("graph", template.graph());
        payload.put("schemaInputs", template.schemaInputs());
        payload.put("schemaOutputs", template.schemaOutputs());
        payload.put("uiState", template.uiState());
        payload.put("currentVersion", template.currentVersion());
        payload.put("createdAt", template.base().createdAt());
        payload.put("updatedAt", template.base().updatedAt());
        return payload;
    }

    private Map<String, Object> copyObjectMap(Map<?, ?> source) {
        Map<String, Object> target = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            target.put(String.valueOf(entry.getKey()), copyValue(entry.getValue()));
        }
        return target;
    }

    private Object copyValue(Object value) {
        if (value instanceof Map<?, ?> mapValue) {
            return copyObjectMap(mapValue);
        }
        if (value instanceof List<?> listValue) {
            java.util.ArrayList<Object> copied = new java.util.ArrayList<>(listValue.size());
            for (Object item : listValue) {
                copied.add(copyValue(item));
            }
            return copied;
        }
        return value;
    }
}
