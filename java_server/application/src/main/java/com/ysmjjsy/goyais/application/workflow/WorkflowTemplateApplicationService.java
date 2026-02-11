/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow template application service for read APIs and command-sugar writes.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.command.CommandApplicationService;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplateCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplatePatchRequest;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.springframework.stereotype.Service;

/**
 * Coordinates workflow template read APIs and command-first write APIs.
 */
@Service
public final class WorkflowTemplateApplicationService {
    private final CommandApplicationService commandService;
    private final WorkflowTemplateRepository templateRepository;

    /**
     * Creates workflow template application service with dependencies.
     * @param commandService TODO
     * @param templateRepository TODO
     */
    public WorkflowTemplateApplicationService(
            CommandApplicationService commandService,
            WorkflowTemplateRepository templateRepository
    ) {
        this.commandService = commandService;
        this.templateRepository = templateRepository;
    }

    /**
     * Creates one draft workflow template through command-first path.
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<WorkflowTemplate> create(WorkflowTemplateCreateRequest request, ExecutionContext context) {
        if (request == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("name", request.name());
        payload.put("description", request.description());
        payload.put("graph", request.graph());
        payload.put("schemaInputs", request.schemaInputs());
        payload.put("schemaOutputs", request.schemaOutputs());
        payload.put("uiState", request.uiState());
        if (request.visibility() != null) {
            payload.put("visibility", request.visibility().name());
        }

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("workflow.createDraft", Map.copyOf(payload), request.visibility()),
                context
        );

        WorkflowTemplate template = readTemplateFromCommand(commandResponse.resource());
        return new WriteResponse<>(template, commandResponse.commandRef());
    }

    /**
     * Patches one workflow template through command-first path.
     * @param templateId TODO
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<WorkflowTemplate> patch(
            String templateId,
            WorkflowTemplatePatchRequest request,
            ExecutionContext context
    ) {
        if (request == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> patch = new LinkedHashMap<>();
        if (request.graph() != null) {
            patch.put("graph", request.graph());
        }
        if (request.operations() != null) {
            patch.put("operations", request.operations());
        }

        Map<String, Object> payload = Map.of(
                "templateId", templateId,
                "patch", patch
        );

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("workflow.patch", payload, null),
                context
        );
        WorkflowTemplate template = readTemplateFromCommand(commandResponse.resource());
        return new WriteResponse<>(template, commandResponse.commandRef());
    }

    /**
     * Publishes one workflow template through command-first path.
     * @param templateId TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<WorkflowTemplate> publish(String templateId, ExecutionContext context) {
        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("workflow.publish", Map.of("templateId", templateId), null),
                context
        );
        WorkflowTemplate template = readTemplateFromCommand(commandResponse.resource());
        return new WriteResponse<>(template, commandResponse.commandRef());
    }

    /**
     * Returns one readable workflow template by id.
     * @param templateId TODO
     * @param context TODO
     * @return TODO
     */
    public WorkflowTemplate get(String templateId, ExecutionContext context) {
        return templateRepository.findReadableById(templateId, context);
    }

    /**
     * Returns readable workflow template list.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    public List<WorkflowTemplate> list(ExecutionContext context, int page, int pageSize) {
        return templateRepository.listReadable(context, normalizePage(page), normalizePageSize(pageSize));
    }

    /**
     * Returns count of readable workflow templates.
     * @param context TODO
     * @return TODO
     */
    public long count(ExecutionContext context) {
        return templateRepository.countReadable(context);
    }

    private int normalizePage(int page) {
        return page <= 0 ? 1 : page;
    }

    private int normalizePageSize(int pageSize) {
        if (pageSize <= 0) {
            return 20;
        }
        return Math.min(pageSize, 200);
    }

    private WorkflowTemplate readTemplateFromCommand(CommandResource command) {
        if (command == null || command.result() == null || !(command.result().get("template") instanceof Map<?, ?> templateRaw)) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return WorkflowContractMapper.toWorkflowTemplate(toStringMap(templateRaw));
    }

    private Map<String, Object> toStringMap(Map<?, ?> source) {
        Map<String, Object> target = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            target.put(String.valueOf(entry.getKey()), entry.getValue());
        }
        return Map.copyOf(target);
    }
}
