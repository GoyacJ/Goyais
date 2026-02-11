/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Applies workflow graph patch operations for workflow.patch command.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.common.ContractException;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Applies graph patch operations to keep Java behavior aligned with Go workflow.patch semantics.
 */
public final class WorkflowPatchApplier {
    private WorkflowPatchApplier() {
    }

    /**
     * Applies patch operations and returns a new graph payload map.
     * @param baseGraph TODO
     * @param operations TODO
     * @return TODO
     */
    public static Map<String, Object> apply(Map<String, Object> baseGraph, List<Map<String, Object>> operations) {
        if (operations == null || operations.isEmpty()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> graph = baseGraph == null ? new LinkedHashMap<>() : deepCopyObjectMap(baseGraph);
        List<Map<String, Object>> nodes = readObjectArray(graph.get("nodes"));
        List<Map<String, Object>> edges = readObjectArray(graph.get("edges"));

        for (Map<String, Object> operation : operations) {
            applyOperation(nodes, edges, operation);
        }

        graph.put("nodes", nodes);
        graph.put("edges", edges);
        return graph;
    }

    private static void applyOperation(
            List<Map<String, Object>> nodes,
            List<Map<String, Object>> edges,
            Map<String, Object> operation
    ) {
        String type = readString(operation, "op").toLowerCase();
        if (type.isBlank()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        String path = readString(operation, "path");
        Map<String, Object> value = readObject(operation, "value");

        switch (type) {
            case "add_node" -> addNode(nodes, value);
            case "update_node" -> updateNode(nodes, path, value);
            case "remove_node" -> removeNode(nodes, edges, path, value);
            case "add_edge" -> addEdge(nodes, edges, value);
            case "remove_edge" -> removeEdge(edges, path, value);
            case "annotate" -> {
                // annotate modifies metadata only and leaves graph topology unchanged.
            }
            default -> throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
    }

    private static void addNode(List<Map<String, Object>> nodes, Map<String, Object> value) {
        if (value == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        String nodeId = readString(value, "id");
        if (nodeId.isBlank() || findNodeIndex(nodes, nodeId) >= 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        nodes.add(deepCopyObjectMap(value));
    }

    private static void updateNode(List<Map<String, Object>> nodes, String path, Map<String, Object> value) {
        if (value == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        String nodeId = readString(value, "id");
        if (nodeId.isBlank()) {
            nodeId = extractPathId(path, "/nodes/");
        }
        int index = findNodeIndex(nodes, nodeId);
        if (index < 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> merged = deepCopyObjectMap(nodes.get(index));
        merged.putAll(value);
        merged.put("id", nodeId);
        nodes.set(index, merged);
    }

    private static void removeNode(
            List<Map<String, Object>> nodes,
            List<Map<String, Object>> edges,
            String path,
            Map<String, Object> value
    ) {
        String nodeId = readString(value, "id");
        if (nodeId.isBlank()) {
            nodeId = extractPathId(path, "/nodes/");
        }
        int index = findNodeIndex(nodes, nodeId);
        if (index < 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        final String removedNodeId = nodeId;
        nodes.remove(index);
        edges.removeIf(edge -> {
            String source = readString(edge, "source");
            if (source.isBlank()) {
                source = readString(edge, "from");
            }
            String target = readString(edge, "target");
            if (target.isBlank()) {
                target = readString(edge, "to");
            }
            return removedNodeId.equals(source) || removedNodeId.equals(target);
        });
    }

    private static void addEdge(
            List<Map<String, Object>> nodes,
            List<Map<String, Object>> edges,
            Map<String, Object> value
    ) {
        if (value == null) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        String source = readString(value, "source");
        if (source.isBlank()) {
            source = readString(value, "from");
        }
        String target = readString(value, "target");
        if (target.isBlank()) {
            target = readString(value, "to");
        }
        if (source.isBlank() || target.isBlank()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        if (findNodeIndex(nodes, source) < 0 || findNodeIndex(nodes, target) < 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        Map<String, Object> edge = deepCopyObjectMap(value);
        String edgeId = readString(edge, "id");
        if (edgeId.isBlank()) {
            edgeId = "e_" + source + "_" + target;
            edge.put("id", edgeId);
        }
        if (findEdgeIndex(edges, edgeId) >= 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        edges.add(edge);
    }

    private static void removeEdge(List<Map<String, Object>> edges, String path, Map<String, Object> value) {
        String edgeId = readString(value, "id");
        if (edgeId.isBlank()) {
            edgeId = extractPathId(path, "/edges/");
        }

        if (!edgeId.isBlank()) {
            int index = findEdgeIndex(edges, edgeId);
            if (index < 0) {
                throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
            }
            edges.remove(index);
            return;
        }

        String source = readString(value, "source");
        if (source.isBlank()) {
            source = readString(value, "from");
        }
        String target = readString(value, "target");
        if (target.isBlank()) {
            target = readString(value, "to");
        }
        if (source.isBlank() || target.isBlank()) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }

        int index = findEdgeByEndpoints(edges, source, target);
        if (index < 0) {
            throw ContractException.of(400, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request");
        }
        edges.remove(index);
    }

    private static int findNodeIndex(List<Map<String, Object>> nodes, String nodeId) {
        for (int index = 0; index < nodes.size(); index += 1) {
            if (nodeId.equals(readString(nodes.get(index), "id"))) {
                return index;
            }
        }
        return -1;
    }

    private static int findEdgeIndex(List<Map<String, Object>> edges, String edgeId) {
        for (int index = 0; index < edges.size(); index += 1) {
            if (edgeId.equals(readString(edges.get(index), "id"))) {
                return index;
            }
        }
        return -1;
    }

    private static int findEdgeByEndpoints(List<Map<String, Object>> edges, String source, String target) {
        for (int index = 0; index < edges.size(); index += 1) {
            Map<String, Object> edge = edges.get(index);
            String edgeSource = readString(edge, "source");
            if (edgeSource.isBlank()) {
                edgeSource = readString(edge, "from");
            }
            String edgeTarget = readString(edge, "target");
            if (edgeTarget.isBlank()) {
                edgeTarget = readString(edge, "to");
            }
            if (source.equals(edgeSource) && target.equals(edgeTarget)) {
                return index;
            }
        }
        return -1;
    }

    private static String extractPathId(String path, String prefix) {
        if (path == null || path.isBlank() || prefix == null || prefix.isBlank()) {
            return "";
        }
        if (!path.startsWith(prefix)) {
            return "";
        }
        String value = path.substring(prefix.length());
        if (value.endsWith("/")) {
            value = value.substring(0, value.length() - 1);
        }
        return value;
    }

    private static List<Map<String, Object>> readObjectArray(Object raw) {
        if (!(raw instanceof List<?> list)) {
            return new ArrayList<>();
        }
        List<Map<String, Object>> result = new ArrayList<>();
        for (Object item : list) {
            if (item instanceof Map<?, ?> map) {
                result.add(deepCopyObjectMap(map));
            }
        }
        return result;
    }

    private static Map<String, Object> readObject(Map<String, Object> source, String key) {
        if (source == null || source.get(key) == null) {
            return null;
        }
        Object value = source.get(key);
        if (!(value instanceof Map<?, ?> map)) {
            return null;
        }
        return deepCopyObjectMap(map);
    }

    private static String readString(Map<String, Object> source, String key) {
        if (source == null || source.get(key) == null) {
            return "";
        }
        return String.valueOf(source.get(key)).trim();
    }

    private static Map<String, Object> deepCopyObjectMap(Map<?, ?> source) {
        Map<String, Object> target = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            target.put(String.valueOf(entry.getKey()), deepCopyValue(entry.getValue()));
        }
        return target;
    }

    private static Object deepCopyValue(Object value) {
        if (value instanceof Map<?, ?> map) {
            return deepCopyObjectMap(map);
        }
        if (value instanceof List<?> list) {
            List<Object> copied = new ArrayList<>(list.size());
            for (Object item : list) {
                copied.add(deepCopyValue(item));
            }
            return copied;
        }
        return value;
    }
}
