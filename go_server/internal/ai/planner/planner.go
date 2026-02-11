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
	"regexp"
	"strings"
)

var ErrInvalidIntent = errors.New("invalid ai intent")

var (
	workflowTemplateIDPattern = regexp.MustCompile(`(?i)\b(?:tpl|template|workflow|wf)_[a-z0-9_-]+\b`)
	workflowRunIDPattern      = regexp.MustCompile(`(?i)\brun_[a-z0-9_-]+\b`)
	algorithmIDPattern        = regexp.MustCompile(`(?i)\b(?:algo|algorithm)_[a-z0-9_-]+\b`)
	scopeIDPattern            = regexp.MustCompile(`(?i)\b(?:workspace|ws|session|run)_[a-z0-9_-]+\b`)
)

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

	if naturalPlan, matched := parseNaturalLanguageIntent(req.Message, tokens); matched {
		return naturalPlan, nil
	}

	return buildRejectPlan(req.Message, tokens), nil
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

func parseNaturalLanguageIntent(rawMessage string, tokens []string) (Plan, bool) {
	detectors := []func(string, []string) (Plan, bool){
		detectNaturalWorkflowPatchIntent,
		detectNaturalWorkflowRetryIntent,
		detectNaturalWorkflowCancelIntent,
		detectNaturalWorkflowRunIntent,
		detectNaturalAlgorithmRunIntent,
		detectNaturalContextBundleRebuildIntent,
	}
	for _, detector := range detectors {
		if plan, matched := detector(rawMessage, tokens); matched {
			if plan.Explainability == nil {
				plan.Explainability = map[string]any{}
			}
			if _, ok := plan.Explainability["tokens"]; !ok {
				plan.Explainability["tokens"] = append([]string{}, tokens...)
			}
			return plan, true
		}
	}
	return Plan{}, false
}

func detectNaturalWorkflowRunIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"run", "start", "execute", "launch", "trigger", "运行", "执行", "启动", "触发"})
	domainSignals := collectSignals(rawMessage, []string{"workflow", "pipeline", "工作流", "流程"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	templateID := extractWorkflowTemplateID(rawMessage, tokens)
	if templateID == "" {
		return Plan{
			Planner:     "workflow.run.nl",
			Reason:      "missing_workflow_template_id_natural",
			Suggestions: []string{"run workflow <templateId>", "workflow.run <templateId>"},
			Explainability: map[string]any{
				"strategy":   "natural_language_missing_arg",
				"confidence": 0.48,
				"signals":    append(actionSignals, domainSignals...),
				"hint":       "provide workflow template id (e.g. tpl_demo)",
			},
		}, true
	}

	plan := buildWorkflowRunPlan(templateID)
	plan.Planner = "workflow.run.nl"
	plan.Reason = "matched_workflow_run_natural"
	plan.Explainability = mergeExplainability(plan.Explainability, map[string]any{
		"strategy":   "natural_language",
		"confidence": 0.74,
		"signals":    append(actionSignals, domainSignals...),
		"extracted": map[string]any{
			"templateId": templateID,
		},
	})
	return plan, true
}

func detectNaturalWorkflowRetryIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"retry", "rerun", "重试", "重跑", "再试"})
	domainSignals := collectSignals(rawMessage, []string{"workflow", "pipeline", "工作流", "流程"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	runID := extractWorkflowRunID(rawMessage, tokens)
	if runID == "" {
		return Plan{
			Planner:     "workflow.retry.nl",
			Reason:      "missing_workflow_run_id_natural",
			Suggestions: []string{"retry workflow <runId>", "workflow.retry <runId>"},
			Explainability: map[string]any{
				"strategy":   "natural_language_missing_arg",
				"confidence": 0.46,
				"signals":    append(actionSignals, domainSignals...),
				"hint":       "provide workflow run id (e.g. run_demo)",
			},
		}, true
	}

	plan := buildWorkflowRetryPlan(runID)
	plan.Planner = "workflow.retry.nl"
	plan.Reason = "matched_workflow_retry_natural"
	plan.Explainability = mergeExplainability(plan.Explainability, map[string]any{
		"strategy":   "natural_language",
		"confidence": 0.71,
		"signals":    append(actionSignals, domainSignals...),
		"extracted": map[string]any{
			"runId": runID,
		},
	})
	return plan, true
}

func detectNaturalWorkflowCancelIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"cancel", "stop", "abort", "terminate", "取消", "停止", "中止"})
	domainSignals := collectSignals(rawMessage, []string{"workflow", "pipeline", "工作流", "流程"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	runID := extractWorkflowRunID(rawMessage, tokens)
	if runID == "" {
		return Plan{
			Planner:     "workflow.cancel.nl",
			Reason:      "missing_workflow_run_id_natural",
			Suggestions: []string{"cancel workflow <runId>", "workflow.cancel <runId>"},
			Explainability: map[string]any{
				"strategy":   "natural_language_missing_arg",
				"confidence": 0.45,
				"signals":    append(actionSignals, domainSignals...),
				"hint":       "provide workflow run id (e.g. run_demo)",
			},
		}, true
	}

	plan := buildWorkflowCancelPlan(runID)
	plan.Planner = "workflow.cancel.nl"
	plan.Reason = "matched_workflow_cancel_natural"
	plan.Explainability = mergeExplainability(plan.Explainability, map[string]any{
		"strategy":   "natural_language",
		"confidence": 0.72,
		"signals":    append(actionSignals, domainSignals...),
		"extracted": map[string]any{
			"runId": runID,
		},
	})
	return plan, true
}

func detectNaturalWorkflowPatchIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"patch", "edit", "update", "modify", "修改", "编辑", "更新", "补丁"})
	domainSignals := collectSignals(rawMessage, []string{"workflow", "canvas", "graph", "工作流", "画布", "流程图"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	templateID := extractWorkflowTemplateID(rawMessage, tokens)
	if templateID == "" {
		return Plan{
			Planner:     "workflow.patch.nl",
			Reason:      "missing_workflow_template_id_natural",
			Suggestions: []string{"patch workflow <templateId> {\"operations\":[...]}"},
			Explainability: map[string]any{
				"strategy":   "natural_language_missing_arg",
				"confidence": 0.47,
				"signals":    append(actionSignals, domainSignals...),
				"hint":       "provide workflow template id (e.g. tpl_demo)",
			},
		}, true
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
			Planner:     "workflow.patch.nl",
			Reason:      "matched_workflow_patch_natural",
			Explainability: map[string]any{
				"strategy":   "natural_language_json_patch",
				"confidence": 0.76,
				"signals":    append(actionSignals, domainSignals...),
				"impact":     impact,
			},
		}, true
	}

	sourceNodeID := extractSourceNodeHint(tokens)
	if sourceNodeID == "" {
		sourceNodeID = extractSourceNodeHintFromMessage(rawMessage)
	}
	plan := buildGeneratedWorkflowPatchPlan(
		templateID,
		rawMessage,
		sourceNodeID,
		"workflow.patch.nl",
		"matched_workflow_patch_generated_natural",
		"natural_language_controlled_patch_generation",
		append(actionSignals, domainSignals...),
		0.63,
	)
	return plan, true
}

func detectNaturalAlgorithmRunIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"run", "execute", "infer", "运行", "执行", "推理"})
	domainSignals := collectSignals(rawMessage, []string{"algorithm", "model", "算法", "模型"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	algorithmID := extractAlgorithmID(rawMessage, tokens)
	if algorithmID == "" {
		return Plan{
			Planner:     "algorithm.run.nl",
			Reason:      "missing_algorithm_id_natural",
			Suggestions: []string{"run algorithm <algorithmId>", "algorithm.run <algorithmId>"},
			Explainability: map[string]any{
				"strategy":   "natural_language_missing_arg",
				"confidence": 0.45,
				"signals":    append(actionSignals, domainSignals...),
				"hint":       "provide algorithm id (e.g. algo_demo)",
			},
		}, true
	}

	plan := buildAlgorithmRunPlan(algorithmID)
	plan.Planner = "algorithm.run.nl"
	plan.Reason = "matched_algorithm_run_natural"
	plan.Explainability = mergeExplainability(plan.Explainability, map[string]any{
		"strategy":   "natural_language",
		"confidence": 0.71,
		"signals":    append(actionSignals, domainSignals...),
		"extracted": map[string]any{
			"algorithmId": algorithmID,
		},
	})
	return plan, true
}

func detectNaturalContextBundleRebuildIntent(rawMessage string, tokens []string) (Plan, bool) {
	actionSignals := collectSignals(rawMessage, []string{"rebuild", "refresh", "regenerate", "重建", "刷新", "重算"})
	domainSignals := collectSignals(rawMessage, []string{"context", "bundle", "contextbundle", "上下文", "记忆"})
	if len(actionSignals) == 0 || len(domainSignals) == 0 {
		return Plan{}, false
	}

	scopeType := "workspace"
	if len(collectSignals(rawMessage, []string{"session", "会话"})) > 0 {
		scopeType = "session"
	}
	if len(collectSignals(rawMessage, []string{"workspace", "工作区"})) > 0 {
		scopeType = "workspace"
	}
	if len(collectSignals(rawMessage, []string{" run ", "运行记录"})) > 0 {
		scopeType = "run"
	}
	scopeID := extractContextScopeID(rawMessage, tokens)

	plan := buildContextBundleRebuildPlan(scopeType, scopeID)
	plan.Planner = "context.bundle.rebuild.nl"
	plan.Reason = "matched_context_bundle_rebuild_natural"
	plan.Explainability = mergeExplainability(plan.Explainability, map[string]any{
		"strategy":   "natural_language",
		"confidence": 0.66,
		"signals":    append(actionSignals, domainSignals...),
		"extracted": map[string]any{
			"scopeType": scopeType,
			"scopeId":   scopeID,
		},
	})
	return plan, true
}

func buildRejectPlan(rawMessage string, tokens []string) Plan {
	workflowSignals := collectSignals(rawMessage, []string{"workflow", "pipeline", "工作流", "流程"})
	if len(workflowSignals) > 0 {
		return buildDomainHintRejectPlan("workflow", "ambiguous_workflow_intent", workflowSuggestions(), workflowSignals, tokens)
	}
	algorithmSignals := collectSignals(rawMessage, []string{"algorithm", "model", "算法", "模型"})
	if len(algorithmSignals) > 0 {
		return buildDomainHintRejectPlan("algorithm", "ambiguous_algorithm_intent", algorithmSuggestions(), algorithmSignals, tokens)
	}
	contextSignals := collectSignals(rawMessage, []string{"context", "bundle", "contextbundle", "上下文", "记忆"})
	if len(contextSignals) > 0 {
		return buildDomainHintRejectPlan("context_bundle", "ambiguous_context_bundle_intent", contextBundleSuggestions(), contextSignals, tokens)
	}
	actionSignals := collectSignals(rawMessage, []string{"run", "start", "execute", "retry", "cancel", "patch", "rebuild", "运行", "执行", "重试", "取消", "修改", "重建"})
	if len(actionSignals) > 0 {
		return Plan{
			Planner:     "reject.missing_domain",
			Reason:      "missing_target_domain",
			Suggestions: defaultSuggestions(),
			Explainability: map[string]any{
				"strategy":   "reject_missing_domain",
				"confidence": 0.28,
				"signals":    actionSignals,
				"tokens":     append([]string{}, tokens...),
			},
		}
	}
	return Plan{
		Planner:     "none",
		Reason:      "unsupported_intent",
		Suggestions: defaultSuggestions(),
		Explainability: map[string]any{
			"strategy":   "reject",
			"confidence": 0.15,
			"tokens":     append([]string{}, tokens...),
		},
	}
}

func buildDomainHintRejectPlan(domain string, reason string, suggestions []string, signals []string, tokens []string) Plan {
	return Plan{
		Planner:     "reject." + domain,
		Reason:      reason,
		Suggestions: append([]string{}, suggestions...),
		Explainability: map[string]any{
			"strategy":   "reject_with_domain_hint",
			"domain":     domain,
			"confidence": 0.34,
			"signals":    append([]string{}, signals...),
			"tokens":     append([]string{}, tokens...),
		},
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
	return buildGeneratedWorkflowPatchPlan(
		templateID,
		rawMessage,
		sourceNodeID,
		"workflow.patch",
		"matched_workflow_patch_generated",
		"controlled_patch_generation",
		nil,
		0.78,
	), true
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

func buildGeneratedWorkflowPatchPlan(
	templateID string,
	rawMessage string,
	sourceNodeID string,
	planner string,
	reason string,
	strategy string,
	signals []string,
	confidence float64,
) Plan {
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
	explainability := map[string]any{
		"strategy":     strategy,
		"sourceNodeId": sourceNodeID,
		"generatedNode": map[string]any{
			"id":   nodeID,
			"type": nodeType,
		},
		"impact": impact,
	}
	if len(signals) > 0 {
		explainability["signals"] = append([]string{}, signals...)
	}
	if confidence > 0 {
		explainability["confidence"] = confidence
	}
	return Plan{
		CommandType:    "workflow.patch",
		Payload:        payload,
		Planner:        planner,
		Reason:         reason,
		Explainability: explainability,
	}
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
	suggestions := make([]string, 0, len(workflowSuggestions())+len(algorithmSuggestions())+len(contextBundleSuggestions()))
	suggestions = append(suggestions, workflowSuggestions()...)
	suggestions = append(suggestions, algorithmSuggestions()...)
	suggestions = append(suggestions, contextBundleSuggestions()...)
	return suggestions
}

func workflowSuggestions() []string {
	return []string{
		"run workflow <templateId>",
		"retry workflow <runId>",
		"cancel workflow <runId>",
		"patch workflow <templateId> {\"operations\":[...]}",
	}
}

func algorithmSuggestions() []string {
	return []string{
		"run algorithm <algorithmId>",
	}
}

func contextBundleSuggestions() []string {
	return []string{
		"context.bundle.rebuild [scopeType] [scopeId]",
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

func collectSignals(rawMessage string, keywords []string) []string {
	message := strings.ToLower(strings.TrimSpace(rawMessage))
	if message == "" || len(keywords) == 0 {
		return nil
	}
	out := make([]string, 0, len(keywords))
	dedup := map[string]struct{}{}
	for _, keyword := range keywords {
		key := strings.ToLower(strings.TrimSpace(keyword))
		if key == "" {
			continue
		}
		if strings.Contains(message, key) {
			if _, exists := dedup[key]; exists {
				continue
			}
			dedup[key] = struct{}{}
			out = append(out, keyword)
		}
	}
	return out
}

func mergeExplainability(base map[string]any, extra map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func extractWorkflowTemplateID(rawMessage string, tokens []string) string {
	if candidate := cleanToken(workflowTemplateIDPattern.FindString(rawMessage)); candidate != "" {
		return candidate
	}
	if candidate := extractIdentifierAfterMarkers(tokens, []string{"workflow", "template", "templateid", "template_id", "工作流", "模板"}); candidate != "" {
		return candidate
	}
	return extractKeyValueIdentifier(rawMessage, []string{"workflow", "template", "workflowId", "templateId", "workflow_id", "template_id"})
}

func extractWorkflowRunID(rawMessage string, tokens []string) string {
	if candidate := cleanToken(workflowRunIDPattern.FindString(rawMessage)); candidate != "" {
		return candidate
	}
	if candidate := extractIdentifierAfterMarkers(tokens, []string{"run", "runid", "run_id", "workflowrun", "运行", "任务"}); candidate != "" {
		return candidate
	}
	return extractKeyValueIdentifier(rawMessage, []string{"run", "runId", "run_id", "workflowRunId", "workflow_run_id"})
}

func extractAlgorithmID(rawMessage string, tokens []string) string {
	if candidate := cleanToken(algorithmIDPattern.FindString(rawMessage)); candidate != "" {
		return candidate
	}
	if candidate := extractIdentifierAfterMarkers(tokens, []string{"algorithm", "algo", "algorithmid", "algorithm_id", "算法"}); candidate != "" {
		return candidate
	}
	return extractKeyValueIdentifier(rawMessage, []string{"algorithm", "algo", "algorithmId", "algorithm_id"})
}

func extractContextScopeID(rawMessage string, tokens []string) string {
	if candidate := cleanToken(scopeIDPattern.FindString(rawMessage)); candidate != "" {
		return candidate
	}
	if candidate := extractIdentifierAfterMarkers(tokens, []string{"workspace", "session", "run", "scope", "工作区", "会话"}); candidate != "" {
		return candidate
	}
	return extractKeyValueIdentifier(rawMessage, []string{"workspace", "session", "run", "scopeId", "scope_id"})
}

func extractIdentifierAfterMarkers(tokens []string, markers []string) string {
	if len(tokens) == 0 || len(markers) == 0 {
		return ""
	}
	markerSet := map[string]struct{}{}
	for _, marker := range markers {
		key := strings.ToLower(strings.TrimSpace(marker))
		if key != "" {
			markerSet[key] = struct{}{}
		}
	}
	for idx := 0; idx < len(tokens); idx++ {
		token := strings.TrimSpace(tokens[idx])
		if token == "" {
			continue
		}
		lowerToken := strings.ToLower(cleanToken(token))
		if _, ok := markerSet[lowerToken]; ok && idx+1 < len(tokens) {
			candidate := cleanToken(tokens[idx+1])
			if isLikelyIdentifier(candidate) {
				return candidate
			}
		}
		if cut := strings.IndexAny(token, "=:"); cut > 0 {
			key := strings.ToLower(cleanToken(token[:cut]))
			if _, ok := markerSet[key]; ok {
				candidate := cleanToken(token[cut+1:])
				if isLikelyIdentifier(candidate) {
					return candidate
				}
			}
		}
	}
	return ""
}

func extractKeyValueIdentifier(rawMessage string, keys []string) string {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(key) + `\b\s*[:=]\s*([a-z0-9._-]+)`)
		matches := pattern.FindStringSubmatch(rawMessage)
		if len(matches) != 2 {
			continue
		}
		candidate := cleanToken(matches[1])
		if isLikelyIdentifier(candidate) {
			return candidate
		}
	}
	return ""
}

func isLikelyIdentifier(candidate string) bool {
	value := strings.TrimSpace(candidate)
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	switch lower {
	case "workflow", "pipeline", "run", "retry", "cancel", "patch", "algorithm", "context", "bundle", "工作流", "算法", "上下文":
		return false
	}
	if strings.ContainsAny(lower, "{}[]()") {
		return false
	}
	return true
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

func extractSourceNodeHintFromMessage(rawMessage string) string {
	pattern := regexp.MustCompile(`(?i)\b(?:source|source_node|source-node|from)\b\s*[:=]?\s*([a-z0-9._-]+)`)
	matches := pattern.FindStringSubmatch(rawMessage)
	if len(matches) != 2 {
		return ""
	}
	return cleanToken(matches[1])
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
