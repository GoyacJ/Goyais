/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unit tests for workflow graph patch operation semantics.
 */
package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.common.ContractException;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

class WorkflowPatchApplierTest {

    @Test
    void shouldApplyAddEdgeAndRemoveNodeOperations() {
        Map<String, Object> baseGraph = Map.of(
                "nodes", List.of(Map.of("id", "start"), Map.of("id", "end")),
                "edges", List.of()
        );
        List<Map<String, Object>> operations = List.of(
                Map.of(
                        "op", "add_node",
                        "value", Map.of("id", "task1", "type", "task")
                ),
                Map.of(
                        "op", "add_edge",
                        "value", Map.of("id", "e_start_task1", "source", "start", "target", "task1")
                ),
                Map.of(
                        "op", "remove_node",
                        "value", Map.of("id", "start")
                )
        );

        Map<String, Object> patched = WorkflowPatchApplier.apply(baseGraph, operations);
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> nodes = (List<Map<String, Object>>) patched.get("nodes");
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> edges = (List<Map<String, Object>>) patched.get("edges");

        Assertions.assertEquals(2, nodes.size());
        Assertions.assertTrue(nodes.stream().anyMatch(node -> "task1".equals(node.get("id"))));
        Assertions.assertTrue(nodes.stream().noneMatch(node -> "start".equals(node.get("id"))));
        Assertions.assertTrue(edges.isEmpty());
    }

    @Test
    void shouldRejectEmptyOperations() {
        ContractException ex = Assertions.assertThrows(
                ContractException.class,
                () -> WorkflowPatchApplier.apply(Map.of("nodes", List.of(), "edges", List.of()), List.of())
        );

        Assertions.assertEquals(400, ex.statusCode());
        Assertions.assertEquals("INVALID_WORKFLOW_REQUEST", ex.code());
    }
}
