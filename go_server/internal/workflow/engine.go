// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type executionPlan struct {
	RunStatus     string
	RunOutputs    json.RawMessage
	RunErrorCode  string
	RunMessageKey string
	RunFinished   bool
	Steps         []plannedStep
}

type plannedStep struct {
	Key        string
	Type       string
	Attempt    int
	Status     string
	Input      json.RawMessage
	Output     json.RawMessage
	ErrorCode  string
	MessageKey string
	RetryAfter int
	WillRetry  bool
	Finished   bool
}

type plannedRunEvent struct {
	StepKey   string
	EventType string
	Payload   json.RawMessage
}

type retryPolicy struct {
	MaxAttempts   int
	BaseBackoffMS int
	MaxBackoffMS  int
}

type workflowGraph struct {
	Nodes []workflowGraphNode `json:"nodes"`
	Edges []workflowGraphEdge `json:"edges"`
}

type workflowGraphNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type workflowGraphEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func buildExecutionPlan(
	graphRaw json.RawMessage,
	inputsRaw json.RawMessage,
	mode string,
	fromStepKey string,
	testNode bool,
) (executionPlan, error) {
	nodesInOrder, adjacency, indegree, err := parseGraphTopology(graphRaw)
	if err != nil {
		return executionPlan{}, err
	}

	selected := selectPlannedNodes(nodesInOrder, adjacency, strings.TrimSpace(fromStepKey), testNode)
	if len(selected) == 0 {
		selected = []workflowGraphNode{{ID: "step-1", Type: "noop"}}
	}

	inputs := inputsRaw
	if len(inputs) == 0 {
		inputs = json.RawMessage(`{}`)
	}
	if !isJSONObject(inputs) {
		return executionPlan{}, ErrInvalidRequest
	}

	stepInputs := decodeJSONMap(inputs)
	failStepKey := detectFailStepKey(stepInputs)
	if failStepKey == "" && len(selected) > 0 {
		failStepKey = selected[0].ID
	}
	retry := parseRetryPolicy(stepInputs)

	switch mode {
	case RunModeRunning:
		return buildRunningPlan(selected, stepInputs, indegree, adjacency), nil
	case RunModeFail:
		return buildFailedPlan(selected, stepInputs, failStepKey, retry), nil
	case RunModeRetry:
		return buildSucceededPlan(selected, stepInputs, RunModeRetry), nil
	case "", RunModeSync:
		return buildSucceededPlan(selected, stepInputs, RunModeSync), nil
	default:
		return executionPlan{}, ErrInvalidRequest
	}
}

func parseGraphTopology(
	graphRaw json.RawMessage,
) ([]workflowGraphNode, map[string][]string, map[string]int, error) {
	if len(graphRaw) == 0 {
		return []workflowGraphNode{{ID: "step-1", Type: "noop"}}, map[string][]string{}, map[string]int{"step-1": 0}, nil
	}

	var graph workflowGraph
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		return nil, nil, nil, ErrInvalidRequest
	}
	if len(graph.Nodes) == 0 {
		return []workflowGraphNode{{ID: "step-1", Type: "noop"}}, map[string][]string{}, map[string]int{"step-1": 0}, nil
	}

	nodes := make([]workflowGraphNode, 0, len(graph.Nodes))
	nodeIndex := make(map[string]int, len(graph.Nodes))
	for idx, node := range graph.Nodes {
		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			return nil, nil, nil, ErrInvalidRequest
		}
		if _, exists := nodeIndex[nodeID]; exists {
			return nil, nil, nil, ErrInvalidRequest
		}
		nodeType := strings.TrimSpace(node.Type)
		if nodeType == "" {
			nodeType = "noop"
		}
		nodes = append(nodes, workflowGraphNode{
			ID:   nodeID,
			Type: nodeType,
		})
		nodeIndex[nodeID] = idx
	}

	adjacency := make(map[string][]string, len(nodes))
	indegree := make(map[string]int, len(nodes))
	for _, node := range nodes {
		adjacency[node.ID] = []string{}
		indegree[node.ID] = 0
	}

	for _, edge := range graph.Edges {
		from := strings.TrimSpace(edge.From)
		if from == "" {
			from = strings.TrimSpace(edge.Source)
		}
		to := strings.TrimSpace(edge.To)
		if to == "" {
			to = strings.TrimSpace(edge.Target)
		}
		if from == "" || to == "" {
			continue
		}
		if _, ok := nodeIndex[from]; !ok {
			return nil, nil, nil, ErrInvalidRequest
		}
		if _, ok := nodeIndex[to]; !ok {
			return nil, nil, nil, ErrInvalidRequest
		}
		adjacency[from] = append(adjacency[from], to)
		indegree[to]++
	}

	ordered, err := topologicalOrder(nodes, adjacency, indegree)
	if err != nil {
		return nil, nil, nil, err
	}
	return ordered, adjacency, indegree, nil
}

func topologicalOrder(
	nodes []workflowGraphNode,
	adjacency map[string][]string,
	indegree map[string]int,
) ([]workflowGraphNode, error) {
	workingIndegree := make(map[string]int, len(indegree))
	for key, value := range indegree {
		workingIndegree[key] = value
	}

	indexByID := make(map[string]int, len(nodes))
	for idx, node := range nodes {
		indexByID[node.ID] = idx
	}

	queue := make([]workflowGraphNode, 0, len(nodes))
	for _, node := range nodes {
		if workingIndegree[node.ID] == 0 {
			queue = append(queue, node)
		}
	}
	sort.Slice(queue, func(i, j int) bool {
		return indexByID[queue[i].ID] < indexByID[queue[j].ID]
	})

	ordered := make([]workflowGraphNode, 0, len(nodes))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ordered = append(ordered, current)

		for _, next := range adjacency[current.ID] {
			workingIndegree[next]--
			if workingIndegree[next] == 0 {
				queue = append(queue, workflowGraphNode{
					ID:   next,
					Type: nodes[indexByID[next]].Type,
				})
			}
		}

		sort.Slice(queue, func(i, j int) bool {
			return indexByID[queue[i].ID] < indexByID[queue[j].ID]
		})
	}

	if len(ordered) != len(nodes) {
		return nil, ErrInvalidRequest
	}
	return ordered, nil
}

func selectPlannedNodes(
	nodes []workflowGraphNode,
	adjacency map[string][]string,
	fromStepKey string,
	testNode bool,
) []workflowGraphNode {
	if fromStepKey == "" {
		return nodes
	}

	exists := false
	for _, node := range nodes {
		if node.ID == fromStepKey {
			exists = true
			break
		}
	}
	if !exists {
		return nodes
	}
	if testNode {
		for _, node := range nodes {
			if node.ID == fromStepKey {
				return []workflowGraphNode{node}
			}
		}
		return nodes
	}

	reachable := make(map[string]struct{}, len(nodes))
	stack := []string{fromStepKey}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := reachable[current]; seen {
			continue
		}
		reachable[current] = struct{}{}
		for _, next := range adjacency[current] {
			stack = append(stack, next)
		}
	}

	selected := make([]workflowGraphNode, 0, len(reachable))
	for _, node := range nodes {
		if _, ok := reachable[node.ID]; ok {
			selected = append(selected, node)
		}
	}
	return selected
}

func buildSucceededPlan(
	nodes []workflowGraphNode,
	inputs map[string]any,
	mode string,
) executionPlan {
	steps := make([]plannedStep, 0, len(nodes))
	for _, node := range nodes {
		steps = append(steps, plannedStep{
			Key:     node.ID,
			Type:    node.Type,
			Attempt: 1,
			Status:  StepStatusSucceeded,
			Input:   mustJSONObjectRaw(inputs),
			Output: buildStepExecutionOutput(
				node.ID,
				node.Type,
				mode,
				StepStatusSucceeded,
				1,
				inputs,
				"",
				"",
				"",
				false,
				0,
			),
			Finished: true,
		})
	}

	return executionPlan{
		RunStatus:   RunStatusSucceeded,
		RunOutputs:  buildRunExecutionOutputFromSteps(mode, RunStatusSucceeded, steps, nil),
		RunFinished: true,
		Steps:       steps,
	}
}

func buildFailedPlan(
	nodes []workflowGraphNode,
	inputs map[string]any,
	failStepKey string,
	retry retryPolicy,
) executionPlan {
	if retry.MaxAttempts <= 0 {
		retry.MaxAttempts = 1
	}
	steps := make([]plannedStep, 0, len(nodes)+retry.MaxAttempts-1)
	failed := false
	for _, node := range nodes {
		step := plannedStep{
			Key:     node.ID,
			Type:    node.Type,
			Attempt: 1,
			Input:   mustJSONObjectRaw(inputs),
		}

		switch {
		case !failed && node.ID != failStepKey:
			step.Status = StepStatusSucceeded
			step.Output = buildStepExecutionOutput(
				node.ID,
				node.Type,
				RunModeFail,
				StepStatusSucceeded,
				1,
				inputs,
				"",
				"",
				"",
				false,
				0,
			)
			step.Finished = true
		case !failed && node.ID == failStepKey:
			for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
				willRetry := attempt < retry.MaxAttempts
				retryAfter := 0
				if willRetry {
					retryAfter = computeRetryBackoffMS(retry, attempt)
				}
				stepAttempt := plannedStep{
					Key:        node.ID,
					Type:       node.Type,
					Attempt:    attempt,
					Status:     StepStatusFailed,
					Input:      mustJSONObjectRaw(inputs),
					ErrorCode:  "WORKFLOW_STEP_FAILED",
					MessageKey: "error.workflow.step_failed",
					Output: buildStepExecutionOutput(
						node.ID,
						node.Type,
						RunModeFail,
						StepStatusFailed,
						attempt,
						inputs,
						"WORKFLOW_STEP_FAILED",
						"error.workflow.step_failed",
						"step_execution_failed",
						willRetry,
						retryAfter,
					),
					Finished: true,
				}
				if willRetry {
					stepAttempt.WillRetry = true
					stepAttempt.RetryAfter = retryAfter
				}
				steps = append(steps, stepAttempt)
			}
			failed = true
			continue
		case failed:
			step.Status = StepStatusSkipped
			step.Output = buildStepExecutionOutput(
				node.ID,
				node.Type,
				RunModeFail,
				StepStatusSkipped,
				1,
				inputs,
				"",
				"",
				"dependency_failed",
				false,
				0,
			)
			step.Finished = true
		}

		steps = append(steps, step)
	}

	return executionPlan{
		RunStatus: RunStatusFailed,
		RunOutputs: buildRunExecutionOutputFromSteps(
			RunModeFail,
			RunStatusFailed,
			steps,
			map[string]any{
				"failedStepKey": failStepKey,
				"maxAttempts":   retry.MaxAttempts,
			},
		),
		RunErrorCode:  "WORKFLOW_RUN_FAILED",
		RunMessageKey: "error.workflow.run_failed",
		RunFinished:   true,
		Steps:         steps,
	}
}

func buildRunningPlan(
	nodes []workflowGraphNode,
	inputs map[string]any,
	indegree map[string]int,
	adjacency map[string][]string,
) executionPlan {
	selectedIDs := make(map[string]struct{}, len(nodes))
	selectedIndegree := make(map[string]int, len(nodes))
	for _, node := range nodes {
		selectedIDs[node.ID] = struct{}{}
		selectedIndegree[node.ID] = 0
	}
	for _, node := range nodes {
		for _, next := range adjacency[node.ID] {
			if _, ok := selectedIDs[next]; ok {
				selectedIndegree[next]++
			}
		}
	}

	rootNodes := make(map[string]struct{})
	for _, node := range nodes {
		if selectedIndegree[node.ID] == 0 {
			rootNodes[node.ID] = struct{}{}
		}
	}
	if len(rootNodes) == 0 && len(nodes) > 0 {
		rootNodes[nodes[0].ID] = struct{}{}
	}

	steps := make([]plannedStep, 0, len(nodes))
	for _, node := range nodes {
		step := plannedStep{
			Key:     node.ID,
			Type:    node.Type,
			Attempt: 1,
			Input:   mustJSONObjectRaw(inputs),
		}
		if _, ok := rootNodes[node.ID]; ok {
			step.Status = StepStatusRunning
			step.Output = buildStepExecutionOutput(
				node.ID,
				node.Type,
				RunModeRunning,
				StepStatusRunning,
				1,
				inputs,
				"",
				"",
				"",
				false,
				0,
			)
		} else {
			step.Status = StepStatusPending
			step.Output = buildStepExecutionOutput(
				node.ID,
				node.Type,
				RunModeRunning,
				StepStatusPending,
				1,
				inputs,
				"",
				"",
				"waiting_for_dependencies",
				false,
				0,
			)
		}
		step.Finished = false
		steps = append(steps, step)
	}

	return executionPlan{
		RunStatus:   RunStatusRunning,
		RunOutputs:  buildRunExecutionOutputFromSteps(RunModeRunning, RunStatusRunning, steps, nil),
		RunFinished: false,
		Steps:       steps,
	}
}

func buildExecutionEvents(plan executionPlan) []plannedRunEvent {
	events := make([]plannedRunEvent, 0, len(plan.Steps)*2+2)
	events = append(events, plannedRunEvent{
		EventType: "workflow.run.started",
		Payload:   mustJSONObjectRaw(map[string]any{"status": RunStatusRunning}),
	})

	for _, step := range plan.Steps {
		attempt := step.Attempt
		if attempt <= 0 {
			attempt = 1
		}
		stepBase := map[string]any{
			"stepKey":  step.Key,
			"stepType": step.Type,
			"attempt":  attempt,
		}
		switch step.Status {
		case StepStatusRunning:
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.started",
				Payload:   mustJSONObjectRaw(map[string]any{"stepKey": step.Key, "stepType": step.Type, "attempt": attempt, "status": StepStatusRunning}),
			})
		case StepStatusSucceeded:
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.started",
				Payload:   mustJSONObjectRaw(map[string]any{"stepKey": step.Key, "stepType": step.Type, "attempt": attempt, "status": StepStatusRunning}),
			})
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.succeeded",
				Payload:   mustJSONObjectRaw(map[string]any{"stepKey": step.Key, "stepType": step.Type, "attempt": attempt, "status": StepStatusSucceeded}),
			})
		case StepStatusFailed:
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.started",
				Payload:   mustJSONObjectRaw(map[string]any{"stepKey": step.Key, "stepType": step.Type, "attempt": attempt, "status": StepStatusRunning}),
			})
			failedPayload := map[string]any{
				"stepKey":  step.Key,
				"stepType": step.Type,
				"attempt":  attempt,
				"status":   StepStatusFailed,
			}
			if strings.TrimSpace(step.ErrorCode) != "" {
				failedPayload["errorCode"] = strings.TrimSpace(step.ErrorCode)
			}
			if strings.TrimSpace(step.MessageKey) != "" {
				failedPayload["messageKey"] = strings.TrimSpace(step.MessageKey)
			}
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.failed",
				Payload:   mustJSONObjectRaw(failedPayload),
			})
			if step.WillRetry {
				retryPayload := map[string]any{
					"stepKey":     step.Key,
					"stepType":    step.Type,
					"attempt":     attempt,
					"nextAttempt": attempt + 1,
					"status":      StepStatusPending,
				}
				if step.RetryAfter > 0 {
					retryPayload["backoffMs"] = step.RetryAfter
				}
				events = append(events, plannedRunEvent{
					StepKey:   step.Key,
					EventType: "workflow.step.retry_scheduled",
					Payload:   mustJSONObjectRaw(retryPayload),
				})
			}
		case StepStatusSkipped:
			stepBase["status"] = StepStatusSkipped
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.skipped",
				Payload:   mustJSONObjectRaw(stepBase),
			})
		case StepStatusCanceled:
			stepBase["status"] = StepStatusCanceled
			events = append(events, plannedRunEvent{
				StepKey:   step.Key,
				EventType: "workflow.step.canceled",
				Payload:   mustJSONObjectRaw(stepBase),
			})
		}
	}

	switch plan.RunStatus {
	case RunStatusSucceeded:
		events = append(events, plannedRunEvent{
			EventType: "workflow.run.succeeded",
			Payload:   mustJSONObjectRaw(map[string]any{"status": RunStatusSucceeded}),
		})
	case RunStatusFailed:
		failedPayload := map[string]any{"status": RunStatusFailed}
		if strings.TrimSpace(plan.RunErrorCode) != "" {
			failedPayload["errorCode"] = strings.TrimSpace(plan.RunErrorCode)
		}
		if strings.TrimSpace(plan.RunMessageKey) != "" {
			failedPayload["messageKey"] = strings.TrimSpace(plan.RunMessageKey)
		}
		events = append(events, plannedRunEvent{
			EventType: "workflow.run.failed",
			Payload:   mustJSONObjectRaw(failedPayload),
		})
	}

	return events
}

func detectFailStepKey(inputs map[string]any) string {
	candidates := []string{"failStepKey", "fail_step_key", "fromStepKey", "from_step_key"}
	for _, key := range candidates {
		if value, ok := inputs[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseRetryPolicy(inputs map[string]any) retryPolicy {
	policy := retryPolicy{
		MaxAttempts:   1,
		BaseBackoffMS: 200,
		MaxBackoffMS:  5000,
	}

	readInt := func(value any) (int, bool) {
		switch v := value.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case json.Number:
			n, err := v.Int64()
			if err == nil {
				return int(n), true
			}
		}
		return 0, false
	}

	lookup := func(keys ...string) (int, bool) {
		for _, key := range keys {
			if raw, ok := inputs[key]; ok {
				if value, found := readInt(raw); found {
					return value, true
				}
			}
		}
		return 0, false
	}

	if retryRaw, ok := inputs["retry"].(map[string]any); ok {
		if value, found := readInt(retryRaw["maxAttempts"]); found {
			policy.MaxAttempts = value
		}
		if value, found := readInt(retryRaw["max_attempts"]); found {
			policy.MaxAttempts = value
		}
		if value, found := readInt(retryRaw["baseBackoffMs"]); found {
			policy.BaseBackoffMS = value
		}
		if value, found := readInt(retryRaw["base_backoff_ms"]); found {
			policy.BaseBackoffMS = value
		}
		if value, found := readInt(retryRaw["maxBackoffMs"]); found {
			policy.MaxBackoffMS = value
		}
		if value, found := readInt(retryRaw["max_backoff_ms"]); found {
			policy.MaxBackoffMS = value
		}
	}

	if value, found := lookup("maxRetries", "max_retries", "retryMaxAttempts", "retry_max_attempts"); found {
		policy.MaxAttempts = value
	}
	if value, found := lookup("baseBackoffMs", "base_backoff_ms", "retryBaseBackoffMs", "retry_base_backoff_ms"); found {
		policy.BaseBackoffMS = value
	}
	if value, found := lookup("maxBackoffMs", "max_backoff_ms", "retryMaxBackoffMs", "retry_max_backoff_ms"); found {
		policy.MaxBackoffMS = value
	}

	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.MaxAttempts > 8 {
		policy.MaxAttempts = 8
	}
	if policy.BaseBackoffMS < 0 {
		policy.BaseBackoffMS = 0
	}
	if policy.MaxBackoffMS <= 0 {
		policy.MaxBackoffMS = 5000
	}
	if policy.BaseBackoffMS > policy.MaxBackoffMS {
		policy.BaseBackoffMS = policy.MaxBackoffMS
	}
	return policy
}

func computeRetryBackoffMS(policy retryPolicy, attempt int) int {
	if attempt <= 0 {
		return 0
	}
	backoff := policy.BaseBackoffMS
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= policy.MaxBackoffMS {
			return policy.MaxBackoffMS
		}
	}
	if backoff > policy.MaxBackoffMS {
		return policy.MaxBackoffMS
	}
	if backoff < 0 {
		return 0
	}
	return backoff
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func mustJSONObjectRaw(payload map[string]any) json.RawMessage {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshal json payload: %v", err))
	}
	return raw
}
