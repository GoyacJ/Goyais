/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unit tests for workflow run command constraints and permission checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

class WorkflowRunCommandHandlerTest {

    @Test
    void shouldRejectRunWhenTemplateIsNotPublished() {
        StubWorkflowTemplateRepository templateRepository = new StubWorkflowTemplateRepository();
        templateRepository.template = template("tpl-1", "user-a", "draft");
        StubWorkflowRunRepository runRepository = new StubWorkflowRunRepository();
        WorkflowRunCommandHandler handler = new WorkflowRunCommandHandler(templateRepository, runRepository, true);

        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> handler.execute(
                        new CommandCreateRequest(
                                "workflow.run",
                                Map.of("templateId", "tpl-1", "inputs", Map.of()),
                                Visibility.PRIVATE
                        ),
                        context("user-a")
                )
        );

        Assertions.assertEquals(400, ex.statusCode());
        Assertions.assertEquals("INVALID_WORKFLOW_REQUEST", ex.code());
    }

    @Test
    void shouldDenyRunForNonOwnerWithoutExecutePermission() {
        StubWorkflowTemplateRepository templateRepository = new StubWorkflowTemplateRepository();
        templateRepository.template = template("tpl-1", "owner-a", "published");
        templateRepository.permissionAllowed = false;
        StubWorkflowRunRepository runRepository = new StubWorkflowRunRepository();
        WorkflowRunCommandHandler handler = new WorkflowRunCommandHandler(templateRepository, runRepository, true);

        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> handler.execute(
                        new CommandCreateRequest("workflow.run", Map.of("templateId", "tpl-1", "inputs", Map.of()), null),
                        context("user-b")
                )
        );

        Assertions.assertEquals(403, ex.statusCode());
        Assertions.assertEquals("FORBIDDEN", ex.code());
    }

    @Test
    void shouldCreateRunWithSyncMode() {
        StubWorkflowTemplateRepository templateRepository = new StubWorkflowTemplateRepository();
        templateRepository.template = template("tpl-1", "user-a", "published");
        StubWorkflowRunRepository runRepository = new StubWorkflowRunRepository();
        WorkflowRunCommandHandler handler = new WorkflowRunCommandHandler(templateRepository, runRepository, true);

        Map<String, Object> result = handler.execute(
                new CommandCreateRequest(
                        "workflow.run",
                        Map.of("templateId", "tpl-1", "mode", "sync", "inputs", Map.of("k", "v")),
                        null
                ),
                context("user-a")
        );

        @SuppressWarnings("unchecked")
        Map<String, Object> run = (Map<String, Object>) result.get("run");
        Assertions.assertEquals("succeeded", run.get("status"));
        Assertions.assertEquals("sync", runRepository.lastMode);
    }

    @Test
    void shouldDenyCancelForNonOwnerWithoutExecutePermission() {
        StubWorkflowTemplateRepository templateRepository = new StubWorkflowTemplateRepository();
        StubWorkflowRunRepository runRepository = new StubWorkflowRunRepository();
        runRepository.existingRun = run("run-1", "owner-a", "running");
        runRepository.permissionAllowed = false;
        WorkflowRunCommandHandler handler = new WorkflowRunCommandHandler(templateRepository, runRepository, true);

        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> handler.execute(
                        new CommandCreateRequest("workflow.cancel", Map.of("runId", "run-1"), null),
                        context("user-b")
                )
        );

        Assertions.assertEquals(403, ex.statusCode());
        Assertions.assertEquals("FORBIDDEN", ex.code());
    }

    private ExecutionContext context(String userId) {
        return new ExecutionContext("tenant-a", "workspace-a", userId, Set.of("member"), "v1", "trace-1");
    }

    private WorkflowTemplate template(String id, String ownerId, String status) {
        return new WorkflowTemplate(
                new ResourceBase(
                        id,
                        "tenant-a",
                        "workspace-a",
                        ownerId,
                        Visibility.PRIVATE,
                        List.of(),
                        status,
                        Instant.parse("2026-02-11T00:00:00Z"),
                        Instant.parse("2026-02-11T00:00:00Z")
                ),
                "demo",
                "",
                Map.of("nodes", List.of(Map.of("id", "start")), "edges", List.of()),
                Map.of(),
                Map.of(),
                Map.of(),
                1
        );
    }

    private WorkflowRun run(String id, String ownerId, String status) {
        Instant now = Instant.parse("2026-02-11T00:00:00Z");
        return new WorkflowRun(
                new ResourceBase(
                        id,
                        "tenant-a",
                        "workspace-a",
                        ownerId,
                        Visibility.PRIVATE,
                        List.of(),
                        status,
                        now,
                        now
                ),
                "tpl-1",
                1,
                1,
                null,
                null,
                "trace-1",
                Map.of(),
                Map.of(),
                now,
                null,
                null,
                null
        );
    }

    private static final class StubWorkflowTemplateRepository implements WorkflowTemplateRepository {
        private WorkflowTemplate template;
        private boolean permissionAllowed = true;

        /**
         * Not used in this test suite.
         * @param context TODO
         * @param name TODO
         * @param description TODO
         * @param visibility TODO
         * @param graph TODO
         * @param schemaInputs TODO
         * @param schemaOutputs TODO
         * @param uiState TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate createDraft(
                ExecutionContext context,
                String name,
                String description,
                Visibility visibility,
                Map<String, Object> graph,
                Map<String, Object> schemaInputs,
                Map<String, Object> schemaOutputs,
                Map<String, Object> uiState,
                Instant now
        ) {
            throw new UnsupportedOperationException("createDraft is not used");
        }

        /**
         * Returns stubbed template for access checks.
         * @param templateId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate findByIdInScope(String templateId, ExecutionContext context) {
            return template;
        }

        /**
         * Not used in this test suite.
         * @param templateId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate findReadableById(String templateId, ExecutionContext context) {
            return template;
        }

        /**
         * Not used in this test suite.
         * @param context TODO
         * @param page TODO
         * @param pageSize TODO
         * @return TODO
         */
        @Override
        public List<WorkflowTemplate> listReadable(ExecutionContext context, int page, int pageSize) {
            return template == null ? List.of() : List.of(template);
        }

        /**
         * Not used in this test suite.
         * @param context TODO
         * @return TODO
         */
        @Override
        public long countReadable(ExecutionContext context) {
            return template == null ? 0 : 1;
        }

        /**
         * Not used in this test suite.
         * @param templateId TODO
         * @param context TODO
         * @param graph TODO
         * @param uiState TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate patch(
                String templateId,
                ExecutionContext context,
                Map<String, Object> graph,
                Map<String, Object> uiState,
                Instant now
        ) {
            throw new UnsupportedOperationException("patch is not used");
        }

        /**
         * Not used in this test suite.
         * @param templateId TODO
         * @param context TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate publish(String templateId, ExecutionContext context, Instant now) {
            throw new UnsupportedOperationException("publish is not used");
        }

        /**
         * Returns configured permission decision for execute checks.
         * @param templateId TODO
         * @param context TODO
         * @param permission TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public boolean hasPermission(String templateId, ExecutionContext context, Permission permission, Instant now) {
            return permissionAllowed;
        }
    }

    private static final class StubWorkflowRunRepository implements WorkflowRunRepository {
        private WorkflowRun existingRun;
        private boolean permissionAllowed = true;
        private String lastMode = "";

        /**
         * Creates deterministic run result and records requested mode.
         * @param context TODO
         * @param template TODO
         * @param visibility TODO
         * @param mode TODO
         * @param fromStepKey TODO
         * @param testNode TODO
         * @param inputs TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowRun createRun(
                ExecutionContext context,
                WorkflowTemplate template,
                Visibility visibility,
                String mode,
                String fromStepKey,
                boolean testNode,
                Map<String, Object> inputs,
                Instant now
        ) {
            this.lastMode = mode;
            this.existingRun = new WorkflowRun(
                    new ResourceBase(
                            "run-1",
                            context.tenantId(),
                            context.workspaceId(),
                            context.userId(),
                            visibility == null ? Visibility.PRIVATE : visibility,
                            List.of(),
                            "succeeded",
                            now,
                            now
                    ),
                    template.base().id(),
                    template.currentVersion(),
                    1,
                    null,
                    fromStepKey,
                    context.traceId(),
                    inputs,
                    Map.of("status", "succeeded"),
                    now,
                    now,
                    0L,
                    null
            );
            return existingRun;
        }

        /**
         * Returns stubbed run for cancel and access checks.
         * @param runId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowRun findByIdInScope(String runId, ExecutionContext context) {
            return existingRun;
        }

        /**
         * Not used in this test suite.
         * @param runId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowRun findReadableById(String runId, ExecutionContext context) {
            return existingRun;
        }

        /**
         * Not used in this test suite.
         * @param context TODO
         * @param page TODO
         * @param pageSize TODO
         * @return TODO
         */
        @Override
        public List<WorkflowRun> listReadable(ExecutionContext context, int page, int pageSize) {
            return existingRun == null ? List.of() : List.of(existingRun);
        }

        /**
         * Not used in this test suite.
         * @param context TODO
         * @return TODO
         */
        @Override
        public long countReadable(ExecutionContext context) {
            return existingRun == null ? 0 : 1;
        }

        /**
         * Returns configured permission decision for execute checks.
         * @param runId TODO
         * @param context TODO
         * @param permission TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public boolean hasPermission(String runId, ExecutionContext context, Permission permission, Instant now) {
            return permissionAllowed;
        }

        /**
         * Returns canceled run snapshot.
         * @param runId TODO
         * @param context TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowRun cancelRun(String runId, ExecutionContext context, Instant now) {
            this.existingRun = new WorkflowRun(
                    new ResourceBase(
                            existingRun.base().id(),
                            existingRun.base().tenantId(),
                            existingRun.base().workspaceId(),
                            existingRun.base().ownerId(),
                            existingRun.base().visibility(),
                            List.of(),
                            "canceled",
                            existingRun.base().createdAt(),
                            now
                    ),
                    existingRun.templateId(),
                    existingRun.templateVersion(),
                    existingRun.attempt(),
                    existingRun.retryOfRunId(),
                    existingRun.replayFromStepKey(),
                    existingRun.traceId(),
                    existingRun.inputs(),
                    existingRun.outputs(),
                    existingRun.startedAt(),
                    now,
                    0L,
                    null
            );
            return existingRun;
        }

        /**
         * Not used in this test suite.
         * @param runId TODO
         * @param context TODO
         * @param page TODO
         * @param pageSize TODO
         * @return TODO
         */
        @Override
        public List<StepRun> listSteps(String runId, ExecutionContext context, int page, int pageSize) {
            return List.of();
        }

        /**
         * Not used in this test suite.
         * @param runId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public long countSteps(String runId, ExecutionContext context) {
            return 0;
        }

        /**
         * Not used in this test suite.
         * @param runId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public List<WorkflowRunEvent> listEvents(String runId, ExecutionContext context) {
            return List.of();
        }
    }
}
