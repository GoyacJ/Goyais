// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package planner

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidIntent = errors.New("invalid ai intent")

type TurnRequest struct {
	Message           string
	IntentCommandType string
	IntentPayload     json.RawMessage
}

type Plan struct {
	CommandType    string
	Payload        json.RawMessage
	Planner        string
	Reason         string
	Suggestions    []string
	Explainability map[string]any
}

type parserFn func(tokens []string, raw string) (Plan, bool)

type intentParser struct {
	name  string
	parse parserFn
}

func PlanTurn(req TurnRequest) (Plan, error) {
	explicitType := strings.TrimSpace(req.IntentCommandType)
	if explicitType != "" {
		if strings.HasPrefix(strings.ToLower(explicitType), "ai.") {
			return Plan{}, ErrInvalidIntent
		}
		payload := objectOrDefault(req.IntentPayload)
		if !isJSONObjectRaw(payload) {
			return Plan{}, ErrInvalidIntent
		}
		return Plan{
			CommandType: explicitType,
			Payload:     payload,
			Planner:     "explicit",
			Reason:      "explicit_intent",
			Explainability: map[string]any{
				"strategy": "explicit",
				"source":   "intentCommandType",
			},
		}, nil
	}

	tokens := tokenize(req.Message)
	for _, parser := range defaultParsers() {
		plan, matched := parser.parse(tokens, req.Message)
		if !matched {
			continue
		}
		if strings.TrimSpace(plan.Planner) == "" {
			plan.Planner = parser.name
		}
		if plan.Explainability == nil {
			plan.Explainability = map[string]any{}
		}
		plan.Explainability["parser"] = parser.name
		plan.Explainability["tokens"] = append([]string{}, tokens...)
		return plan, nil
	}

	return Plan{
		Planner:     "none",
		Reason:      "unsupported_intent",
		Suggestions: defaultSuggestions(),
		Explainability: map[string]any{
			"strategy": "reject",
			"tokens":   append([]string{}, tokens...),
		},
	}, nil
}

func defaultParsers() []intentParser {
	return []intentParser{
		{name: "workflow.run", parse: parseWorkflowRunIntent},
		{name: "workflow.retry", parse: parseWorkflowRetryIntent},
		{name: "workflow.cancel", parse: parseWorkflowCancelIntent},
		{name: "workflow.patch", parse: parseWorkflowPatchIntent},
		{name: "algorithm.run", parse: parseAlgorithmRunIntent},
		{name: "context.bundle.rebuild", parse: parseContextBundleRebuildIntent},
	}
}

func parseWorkflowRunIntent(tokens []string, _ string) (Plan, bool) {
	if len(tokens) >= 2 && equalsToken(tokens[0], "run") && equalsToken(tokens[1], "workflow") {
		if len(tokens) < 3 {
			return missingArgPlan("workflow.run", "missing_workflow_template_id", "run workflow <templateId>"), true
		}
		return buildWorkflowRunPlan(cleanToken(tokens[2])), true
	}
	if len(tokens) >= 1 && equalsToken(tokens[0], "workflow.run") {
		if len(tokens) < 2 {
			return missingArgPlan("workflow.run", "missing_workflow_template_id", "workflow.run <templateId>"), true
		}
		return buildWorkflowRunPlan(cleanToken(tokens[1])), true
	}
	return Plan{}, false
}

func parseWorkflowRetryIntent(tokens []string, _ string) (Plan, bool) {
	if len(tokens) >= 2 && equalsToken(tokens[0], "retry") && equalsToken(tokens[1], "workflow") {
		if len(tokens) < 3 {
			return missingArgPlan("workflow.retry", "missing_workflow_run_id", "retry workflow <runId>"), true
		}
		return buildWorkflowRetryPlan(cleanToken(tokens[2])), true
	}
	if len(tokens) >= 1 && equalsToken(tokens[0], "workflow.retry") {
		if len(tokens) < 2 {
			return missingArgPlan("workflow.retry", "missing_workflow_run_id", "workflow.retry <runId>"), true
		}
		return buildWorkflowRetryPlan(cleanToken(tokens[1])), true
	}
	return Plan{}, false
}

func parseWorkflowCancelIntent(tokens []string, _ string) (Plan, bool) {
	if len(tokens) >= 2 && equalsToken(tokens[0], "cancel") && equalsToken(tokens[1], "workflow") {
		if len(tokens) < 3 {
			return missingArgPlan("workflow.cancel", "missing_workflow_run_id", "cancel workflow <runId>"), true
		}
		return buildWorkflowCancelPlan(cleanToken(tokens[2])), true
	}
	if len(tokens) >= 1 && equalsToken(tokens[0], "workflow.cancel") {
		if len(tokens) < 2 {
			return missingArgPlan("workflow.cancel", "missing_workflow_run_id", "workflow.cancel <runId>"), true
		}
		return buildWorkflowCancelPlan(cleanToken(tokens[1])), true
	}
	return Plan{}, false
}

func parseWorkflowPatchIntent(tokens []string, rawMessage string) (Plan, bool) {
	var templateID string
	switch {
	case len(tokens) >= 1 && equalsToken(tokens[0], "workflow.patch"):
		if len(tokens) < 2 {
			return missingArgPlan("workflow.patch", "missing_workflow_template_id", "workflow.patch <templateId> {\"operations\":[...]}"), true
		}
		templateID = cleanToken(tokens[1])
	case len(tokens) >= 2 && equalsToken(tokens[0], "patch") && equalsToken(tokens[1], "workflow"):
		if len(tokens) < 3 {
			return missingArgPlan("workflow.patch", "missing_workflow_template_id", "patch workflow <templateId> {\"operations\":[...]}"), true
		}
		templateID = cleanToken(tokens[2])
	default:
		return Plan{}, false
	}

	if templateID == "" {
		return missingArgPlan("workflow.patch", "missing_workflow_template_id", "patch workflow <templateId> {\"operations\":[...]}"), true
	}

	patchRaw := extractJSONPatchFromMessage(rawMessage)
	if isJSONObjectRaw(patchRaw) {
		payload, _ := json.Marshal(map[string]any{
			"templateId": templateID,
			"patch":      json.RawMessage(patchRaw),
		})
		impact := summarizePatchImpact(patchRaw)
		return Plan{
			CommandType: "workflow.patch",
			Payload:     payload,
			Planner:     "workflow.patch",
			Reason:      "matched_workflow_patch_explicit",
			Explainability: map[string]any{
				"strategy": "explicit_json_patch",
				"impact":   impact,
			},
		}, true
	}

	sourceNodeID := extractSourceNodeHint(tokens)
	nodeType, label, inputType, outputType := inferPatchNodePreset(rawMessage)
	nodeID := deterministicPatchNodeID(templateID, sourceNodeID, rawMessage, nodeType)
	ops := make([]map[string]any, 0, 3)
	ops = append(ops, map[string]any{
		"op": "add_node",
		"value": map[string]any{
			"id":   nodeID,
			"type": nodeType,
			"position": map[string]any{
				"x": 420,
				"y": 220,
			},
			"data": map[string]any{
				"label":      label,
				"nodeType":   nodeType,
				"inputType":  inputType,
				"outputType": outputType,
			},
		},
	})
	if sourceNodeID != "" {
		ops = append(ops, map[string]any{
			"op": "add_edge",
			"value": map[string]any{
				"id":     fmt.Sprintf("e_%s_%s", sourceNodeID, nodeID),
				"source": sourceNodeID,
				"target": nodeID,
			},
		})
	}
	ops = append(ops, map[string]any{
		"op":   "annotate",
		"path": "/ui_state/ai_patch",
		"value": map[string]any{
			"source":       "ai.intent.plan",
			"nodeId":       nodeID,
			"nodeType":     nodeType,
			"sourceNodeId": sourceNodeID,
		},
	})

	patch := map[string]any{"operations": ops}
	patchPayload, _ := json.Marshal(patch)
	payload, _ := json.Marshal(map[string]any{
		"templateId": templateID,
		"patch":      json.RawMessage(patchPayload),
	})
	impact := map[string]any{
		"addedNodes":   1,
		"removedNodes": 0,
		"changedNodes": 0,
		"addedEdges":   boolToInt(sourceNodeID != ""),
		"removedEdges": 0,
	}

	return Plan{
		CommandType: "workflow.patch",
		Payload:     payload,
		Planner:     "workflow.patch",
		Reason:      "matched_workflow_patch_generated",
		Explainability: map[string]any{
			"strategy":     "controlled_patch_generation",
			"sourceNodeId": sourceNodeID,
			"generatedNode": map[string]any{
				"id":   nodeID,
				"type": nodeType,
			},
			"impact": impact,
		},
	}, true
}

func parseAlgorithmRunIntent(tokens []string, _ string) (Plan, bool) {
	if len(tokens) >= 2 && equalsToken(tokens[0], "run") && equalsToken(tokens[1], "algorithm") {
		if len(tokens) < 3 {
			return missingArgPlan("algorithm.run", "missing_algorithm_id", "run algorithm <algorithmId>"), true
		}
		return buildAlgorithmRunPlan(cleanToken(tokens[2])), true
	}
	if len(tokens) >= 1 && equalsToken(tokens[0], "algorithm.run") {
		if len(tokens) < 2 {
			return missingArgPlan("algorithm.run", "missing_algorithm_id", "algorithm.run <algorithmId>"), true
		}
		return buildAlgorithmRunPlan(cleanToken(tokens[1])), true
	}
	return Plan{}, false
}

func parseContextBundleRebuildIntent(tokens []string, _ string) (Plan, bool) {
	if len(tokens) >= 1 && equalsToken(tokens[0], "context.bundle.rebuild") {
		scopeType := "workspace"
		scopeID := ""
		if len(tokens) >= 2 {
			scopeType = cleanToken(tokens[1])
		}
		if len(tokens) >= 3 {
			scopeID = cleanToken(tokens[2])
		}
		return buildContextBundleRebuildPlan(scopeType, scopeID), true
	}
	if len(tokens) >= 3 && equalsToken(tokens[0], "rebuild") && equalsToken(tokens[1], "context") && equalsToken(tokens[2], "bundle") {
		scopeType := "workspace"
		scopeID := ""
		if len(tokens) >= 4 {
			scopeType = cleanToken(tokens[3])
		}
		if len(tokens) >= 5 {
			scopeID = cleanToken(tokens[4])
		}
		return buildContextBundleRebuildPlan(scopeType, scopeID), true
	}
	return Plan{}, false
}

func buildWorkflowRunPlan(templateID string) Plan {
	if templateID == "" {
		return missingArgPlan("workflow.run", "missing_workflow_template_id", "run workflow <templateId>")
	}
	payload, _ := json.Marshal(map[string]any{
		"templateId": templateID,
		"mode":       "sync",
		"inputs":     map[string]any{},
	})
	return Plan{
		CommandType: "workflow.run",
		Payload:     payload,
		Planner:     "workflow.run",
		Reason:      "matched_workflow_run",
		Explainability: map[string]any{
			"strategy": "command_template",
			"target":   templateID,
		},
	}
}

func buildWorkflowRetryPlan(runID string) Plan {
	if runID == "" {
		return missingArgPlan("workflow.retry", "missing_workflow_run_id", "retry workflow <runId>")
	}
	payload, _ := json.Marshal(map[string]any{
		"runId": runID,
		"mode":  "sync",
	})
	return Plan{
		CommandType: "workflow.retry",
		Payload:     payload,
		Planner:     "workflow.retry",
		Reason:      "matched_workflow_retry",
		Explainability: map[string]any{
			"strategy": "command_template",
			"target":   runID,
		},
	}
}

func buildWorkflowCancelPlan(runID string) Plan {
	if runID == "" {
		return missingArgPlan("workflow.cancel", "missing_workflow_run_id", "cancel workflow <runId>")
	}
	payload, _ := json.Marshal(map[string]any{
		"runId": runID,
	})
	return Plan{
		CommandType: "workflow.cancel",
		Payload:     payload,
		Planner:     "workflow.cancel",
		Reason:      "matched_workflow_cancel",
		Explainability: map[string]any{
			"strategy": "command_template",
			"target":   runID,
		},
	}
}

func buildAlgorithmRunPlan(algorithmID string) Plan {
	if algorithmID == "" {
		return missingArgPlan("algorithm.run", "missing_algorithm_id", "run algorithm <algorithmId>")
	}
	payload, _ := json.Marshal(map[string]any{
		"algorithmId": algorithmID,
		"inputs":      map[string]any{},
		"mode":        "sync",
	})
	return Plan{
		CommandType: "algorithm.run",
		Payload:     payload,
		Planner:     "algorithm.run",
		Reason:      "matched_algorithm_run",
		Explainability: map[string]any{
			"strategy": "command_template",
			"target":   algorithmID,
		},
	}
}

func buildContextBundleRebuildPlan(scopeType string, scopeID string) Plan {
	normalizedScopeType := strings.ToLower(strings.TrimSpace(scopeType))
	if normalizedScopeType == "" {
		normalizedScopeType = "workspace"
	}
	scopeID = strings.TrimSpace(scopeID)
	payload, _ := json.Marshal(map[string]any{
		"scopeType":  normalizedScopeType,
		"scopeId":    scopeID,
		"visibility": "PRIVATE",
	})
	return Plan{
		CommandType: "context.bundle.rebuild",
		Payload:     payload,
		Planner:     "context.bundle.rebuild",
		Reason:      "matched_context_bundle_rebuild",
		Explainability: map[string]any{
			"strategy":  "command_template",
			"scopeType": normalizedScopeType,
			"scopeId":   scopeID,
		},
	}
}

func missingArgPlan(planner string, reason string, suggestion string) Plan {
	return Plan{
		Planner:     planner,
		Reason:      reason,
		Suggestions: []string{suggestion},
		Explainability: map[string]any{
			"strategy": "missing_required_argument",
		},
	}
}

func defaultSuggestions() []string {
	return []string{
		"run workflow <templateId>",
		"retry workflow <runId>",
		"cancel workflow <runId>",
		"patch workflow <templateId> {\"operations\":[...]}",
		"run algorithm <algorithmId>",
	}
}

func objectOrDefault(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func tokenize(raw string) []string {
	return strings.Fields(strings.TrimSpace(raw))
}

func equalsToken(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func cleanToken(raw string) string {
	token := strings.TrimSpace(raw)
	token = strings.Trim(token, "`\"'.,;:()[]{}")
	return token
}

func extractJSONPatchFromMessage(raw string) json.RawMessage {
	message := strings.TrimSpace(raw)
	start := strings.Index(message, "{")
	end := strings.LastIndex(message, "}")
	if start < 0 || end <= start {
		return nil
	}
	candidate := json.RawMessage(strings.TrimSpace(message[start : end+1]))
	if !isJSONObjectRaw(candidate) {
		return nil
	}
	return candidate
}

func isJSONObjectRaw(raw json.RawMessage) bool {
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}

func extractSourceNodeHint(tokens []string) string {
	for idx := 0; idx < len(tokens)-1; idx++ {
		if equalsToken(tokens[idx], "from") || equalsToken(tokens[idx], "source") || equalsToken(tokens[idx], "sourceNode") {
			candidate := cleanToken(tokens[idx+1])
			if candidate != "" {
				return candidate
			}
		}
	}
	for _, token := range tokens {
		if strings.HasPrefix(strings.ToLower(token), "source=") {
			candidate := cleanToken(strings.TrimPrefix(token, "source="))
			if candidate != "" {
				return candidate
			}
		}
		if strings.HasPrefix(strings.ToLower(token), "source-node=") {
			candidate := cleanToken(strings.TrimPrefix(token, "source-node="))
			if candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func inferPatchNodePreset(rawMessage string) (nodeType string, label string, inputType string, outputType string) {
	lower := strings.ToLower(rawMessage)
	switch {
	case strings.Contains(lower, "algorithm"):
		return "algorithm.detect", "Algorithm · Detect (AI)", "json", "json"
	case strings.Contains(lower, "transform"):
		return "transform.json", "Transform · JSON (AI)", "text", "json"
	case strings.Contains(lower, "model"):
		return "model.llm", "Model · LLM (AI)", "text", "text"
	case strings.Contains(lower, "tool"):
		return "tool.http", "Tool · HTTP (AI)", "json", "json"
	case strings.Contains(lower, "output"):
		return "output.asset", "Output · Asset (AI)", "json", "any"
	case strings.Contains(lower, "input"):
		return "input.text", "Input · Text (AI)", "none", "text"
	default:
		return "control.branch", "Control · Branch (AI)", "json", "json"
	}
}

func deterministicPatchNodeID(templateID string, sourceNodeID string, message string, nodeType string) string {
	hash := sha1.Sum([]byte(strings.Join([]string{templateID, sourceNodeID, nodeType, strings.TrimSpace(message)}, "|")))
	return "ai_patch_" + hex.EncodeToString(hash[:])[:10]
}

func summarizePatchImpact(patchRaw json.RawMessage) map[string]any {
	impact := map[string]any{
		"addedNodes":   0,
		"removedNodes": 0,
		"changedNodes": 0,
		"addedEdges":   0,
		"removedEdges": 0,
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(patchRaw, &body); err != nil {
		return impact
	}
	opsRaw, ok := body["operations"]
	if !ok {
		if _, hasGraph := body["graph"]; hasGraph {
			impact["changedNodes"] = 1
		}
		return impact
	}
	var ops []map[string]any
	if err := json.Unmarshal(opsRaw, &ops); err != nil {
		return impact
	}
	for _, op := range ops {
		switch strings.ToLower(strings.TrimSpace(readString(op, "op"))) {
		case "add_node":
			impact["addedNodes"] = toInt(impact["addedNodes"]) + 1
		case "remove_node":
			impact["removedNodes"] = toInt(impact["removedNodes"]) + 1
		case "update_node":
			impact["changedNodes"] = toInt(impact["changedNodes"]) + 1
		case "add_edge":
			impact["addedEdges"] = toInt(impact["addedEdges"]) + 1
		case "remove_edge":
			impact["removedEdges"] = toInt(impact["removedEdges"]) + 1
		}
	}
	return impact
}

func readString(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return strings.TrimSpace(value)
}

func toInt(raw any) int {
	value, _ := raw.(int)
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
