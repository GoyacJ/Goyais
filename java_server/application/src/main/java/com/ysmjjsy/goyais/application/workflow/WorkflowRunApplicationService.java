/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Workflow run application service for read APIs and command-sugar writes.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.command.CommandApplicationService;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.springframework.stereotype.Service;

/**
 * Coordinates workflow run read APIs and command-first write APIs.
 */
@Service
public final class WorkflowRunApplicationService {
    private final CommandApplicationService commandService;
    private final WorkflowRunRepository runRepository;

    /**
     * Creates workflow run application service with dependencies.
     * @param commandService TODO
     * @param runRepository TODO
     */
    public WorkflowRunApplicationService(
            CommandApplicationService commandService,
            WorkflowRunRepository runRepository
    ) {
        this.commandService = commandService;
        this.runRepository = runRepository;
    }

    /**
     * Creates one workflow run through command-first path.
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<WorkflowRun> create(WorkflowRunCreateRequest request, ExecutionContext context) {
        if (request == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("templateId", request.templateId());
        if (request.templateVersion() != null) {
            payload.put("templateVersion", request.templateVersion());
        }
        if (request.inputs() != null) {
            payload.put("inputs", request.inputs());
        }
        if (request.mode() != null) {
            payload.put("mode", request.mode());
        }
        if (request.fromStepKey() != null) {
            payload.put("fromStepKey", request.fromStepKey());
        }
        if (request.testNode() != null) {
            payload.put("testNode", request.testNode());
        }
        if (request.visibility() != null) {
            payload.put("visibility", request.visibility().name());
        }

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("workflow.run", Map.copyOf(payload), request.visibility()),
                context
        );

        WorkflowRun run = readRunFromCommand(commandResponse.resource());
        return new WriteResponse<>(run, commandResponse.commandRef());
    }

    /**
     * Cancels one workflow run through command-first path.
     * @param runId TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<WorkflowRun> cancel(String runId, ExecutionContext context) {
        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("workflow.cancel", Map.of("runId", runId), null),
                context
        );
        WorkflowRun run = readRunFromCommand(commandResponse.resource());
        return new WriteResponse<>(run, commandResponse.commandRef());
    }

    /**
     * Returns one readable workflow run by id.
     * @param runId TODO
     * @param context TODO
     * @return TODO
     */
    public WorkflowRun get(String runId, ExecutionContext context) {
        return runRepository.findReadableById(runId, context);
    }

    /**
     * Returns readable workflow run list.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    public List<WorkflowRun> list(ExecutionContext context, int page, int pageSize) {
        return runRepository.listReadable(context, normalizePage(page), normalizePageSize(pageSize));
    }

    /**
     * Returns count of readable workflow runs.
     * @param context TODO
     * @return TODO
     */
    public long count(ExecutionContext context) {
        return runRepository.countReadable(context);
    }

    /**
     * Returns readable step runs for one workflow run.
     * @param runId TODO
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    public List<StepRun> listSteps(String runId, ExecutionContext context, int page, int pageSize) {
        return runRepository.listSteps(runId, context, normalizePage(page), normalizePageSize(pageSize));
    }

    /**
     * Returns count of step runs for one workflow run.
     * @param runId TODO
     * @param context TODO
     * @return TODO
     */
    public long countSteps(String runId, ExecutionContext context) {
        return runRepository.countSteps(runId, context);
    }

    /**
     * Returns workflow run events for one workflow run.
     * @param runId TODO
     * @param context TODO
     * @return TODO
     */
    public List<WorkflowRunEvent> listEvents(String runId, ExecutionContext context) {
        return runRepository.listEvents(runId, context);
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

    private WorkflowRun readRunFromCommand(CommandResource command) {
        if (command == null || command.result() == null || !(command.result().get("run") instanceof Map<?, ?> runRaw)) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return WorkflowContractMapper.toWorkflowRun(toStringMap(runRaw));
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
