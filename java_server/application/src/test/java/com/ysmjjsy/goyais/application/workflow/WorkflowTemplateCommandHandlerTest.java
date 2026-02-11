/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unit tests for workflow template command authorization and patch behavior.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

class WorkflowTemplateCommandHandlerTest {

    @Test
    void shouldRejectPublicVisibilityOnCreateDraft() {
        StubWorkflowTemplateRepository repository = new StubWorkflowTemplateRepository();
        WorkflowTemplateCommandHandler handler = new WorkflowTemplateCommandHandler(repository, true);

        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> handler.execute(
                        new CommandCreateRequest(
                                "workflow.createDraft",
                                Map.of("name", "demo", "graph", Map.of(), "visibility", "PUBLIC"),
                                Visibility.PUBLIC
                        ),
                        context("user-a")
                )
        );

        Assertions.assertEquals(403, ex.statusCode());
        Assertions.assertEquals("FORBIDDEN", ex.code());
    }

    @Test
    void shouldApplyPatchOperationsWhenTemplateOwnerUpdatesGraph() {
        StubWorkflowTemplateRepository repository = new StubWorkflowTemplateRepository();
        repository.template = template("tpl-1", "user-a", "draft", Map.of(
                "nodes", List.of(Map.of("id", "start")),
                "edges", List.of()
        ));
        WorkflowTemplateCommandHandler handler = new WorkflowTemplateCommandHandler(repository, true);

        Map<String, Object> result = handler.execute(
                new CommandCreateRequest(
                        "workflow.patch",
                        Map.of(
                                "templateId", "tpl-1",
                                "patch", Map.of(
                                        "operations", List.of(
                                                Map.of(
                                                        "op", "add_node",
                                                        "value", Map.of("id", "task1")
                                                )
                                        )
                                )
                        ),
                        null
                ),
                context("user-a")
        );

        @SuppressWarnings("unchecked")
        Map<String, Object> templatePayload = (Map<String, Object>) result.get("template");
        @SuppressWarnings("unchecked")
        Map<String, Object> graph = (Map<String, Object>) templatePayload.get("graph");
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> nodes = (List<Map<String, Object>>) graph.get("nodes");

        Assertions.assertEquals("tpl-1", templatePayload.get("id"));
        Assertions.assertTrue(nodes.stream().anyMatch(node -> "task1".equals(node.get("id"))));
    }

    @Test
    void shouldDenyPublishForNonOwnerWithoutManagePermission() {
        StubWorkflowTemplateRepository repository = new StubWorkflowTemplateRepository();
        repository.template = template("tpl-1", "owner-a", "draft", Map.of(
                "nodes", List.of(Map.of("id", "start")),
                "edges", List.of()
        ));
        repository.permissionAllowed = false;
        WorkflowTemplateCommandHandler handler = new WorkflowTemplateCommandHandler(repository, true);

        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> handler.execute(
                        new CommandCreateRequest("workflow.publish", Map.of("templateId", "tpl-1"), null),
                        context("user-b")
                )
        );

        Assertions.assertEquals(403, ex.statusCode());
        Assertions.assertEquals("FORBIDDEN", ex.code());
    }

    private ExecutionContext context(String userId) {
        return new ExecutionContext("tenant-a", "workspace-a", userId, Set.of("member"), "v1", "trace-1");
    }

    private WorkflowTemplate template(String id, String ownerId, String status, Map<String, Object> graph) {
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
                graph,
                Map.of(),
                Map.of(),
                Map.of(),
                1
        );
    }

    private static final class StubWorkflowTemplateRepository implements WorkflowTemplateRepository {
        private WorkflowTemplate template;
        private boolean permissionAllowed = true;

        /**
         * Stores and returns a new draft template.
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
            this.template = new WorkflowTemplate(
                    new ResourceBase(
                            "tpl-1",
                            context.tenantId(),
                            context.workspaceId(),
                            context.userId(),
                            visibility,
                            List.of(),
                            "draft",
                            now,
                            now
                    ),
                    name,
                    description,
                    graph,
                    schemaInputs,
                    schemaOutputs,
                    uiState,
                    1
            );
            return template;
        }

        /**
         * Returns stubbed in-scope template.
         * @param templateId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate findByIdInScope(String templateId, ExecutionContext context) {
            return template;
        }

        /**
         * Returns stubbed readable template.
         * @param templateId TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate findReadableById(String templateId, ExecutionContext context) {
            return template;
        }

        /**
         * Returns singleton readable list.
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
         * Returns readable template count.
         * @param context TODO
         * @return TODO
         */
        @Override
        public long countReadable(ExecutionContext context) {
            return template == null ? 0 : 1;
        }

        /**
         * Updates graph and returns latest template.
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
            this.template = new WorkflowTemplate(
                    new ResourceBase(
                            template.base().id(),
                            template.base().tenantId(),
                            template.base().workspaceId(),
                            template.base().ownerId(),
                            template.base().visibility(),
                            List.of(),
                            template.base().status(),
                            template.base().createdAt(),
                            now
                    ),
                    template.name(),
                    template.description(),
                    graph,
                    template.schemaInputs(),
                    template.schemaOutputs(),
                    uiState,
                    template.currentVersion()
            );
            return template;
        }

        /**
         * Marks template as published and bumps version.
         * @param templateId TODO
         * @param context TODO
         * @param now TODO
         * @return TODO
         */
        @Override
        public WorkflowTemplate publish(String templateId, ExecutionContext context, Instant now) {
            this.template = new WorkflowTemplate(
                    new ResourceBase(
                            template.base().id(),
                            template.base().tenantId(),
                            template.base().workspaceId(),
                            template.base().ownerId(),
                            template.base().visibility(),
                            List.of(),
                            "published",
                            template.base().createdAt(),
                            now
                    ),
                    template.name(),
                    template.description(),
                    template.graph(),
                    template.schemaInputs(),
                    template.schemaOutputs(),
                    template.uiState(),
                    template.currentVersion() + 1
            );
            return template;
        }

        /**
         * Returns configured permission decision for non-owner checks.
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
}
