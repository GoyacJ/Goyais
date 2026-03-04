package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"

	runtimeapplication "goyais/services/hub/internal/runtime/application"
)

func RunGraphHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		runID := strings.TrimSpace(r.PathValue("run_id"))
		if runID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id is required", map[string]any{})
			return
		}

		graph, ok := buildRunTaskGraph(state, r, runID)
		if !ok {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}
		writeJSON(w, http.StatusOK, toHTTPAPIAgentGraph(graph))
	}
}

func RunTasksHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		runID := strings.TrimSpace(r.PathValue("run_id"))
		if runID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id is required", map[string]any{})
			return
		}
		graph, ok := buildRunTaskGraph(state, r, runID)
		if !ok {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}

		stateFilter := strings.TrimSpace(r.URL.Query().Get("state"))
		if stateFilter != "" && !isValidTaskState(TaskState(stateFilter)) {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "state must be one of queued/blocked/running/retrying/completed/failed/cancelled", map[string]any{
				"state": stateFilter,
			})
			return
		}
		filtered := runtimeapplication.FilterRunTasksByState(graph.Tasks, stateFilter)
		tasks := make([]TaskNode, 0, len(filtered))
		for _, item := range filtered {
			tasks = append(tasks, toHTTPAPITaskNode(item))
		}

		start, limit := parseCursorLimit(r)
		paged, next := paginateTaskNodes(tasks, start, limit)
		writeJSON(w, http.StatusOK, RunTaskListResponse{
			Items:      paged,
			NextCursor: next,
		})
	}
}

func RunTaskByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		runID := strings.TrimSpace(r.PathValue("run_id"))
		taskID := strings.TrimSpace(r.PathValue("task_id"))
		if runID == "" || taskID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id and task_id are required", map[string]any{})
			return
		}
		graph, ok := buildRunTaskGraph(state, r, runID)
		if !ok {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}
		task, found := runtimeapplication.FindRunTaskByID(graph.Tasks, taskID)
		if !found {
			WriteStandardError(w, r, http.StatusNotFound, "TASK_NOT_FOUND", "Task does not exist", map[string]any{
				"run_id":  runID,
				"task_id": taskID,
			})
			return
		}
		writeJSON(w, http.StatusOK, toHTTPAPITaskNode(task))
	}
}

func RunTaskControlHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		runID := strings.TrimSpace(r.PathValue("run_id"))
		taskID := strings.TrimSpace(r.PathValue("task_id"))
		if runID == "" || taskID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id and task_id are required", map[string]any{})
			return
		}

		graph, ok := buildRunTaskGraph(state, r, runID)
		if !ok {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}
		if _, found := runtimeapplication.FindRunTaskByID(graph.Tasks, taskID); !found {
			WriteStandardError(w, r, http.StatusNotFound, "TASK_NOT_FOUND", "Task does not exist", map[string]any{
				"run_id":  runID,
				"task_id": taskID,
			})
			return
		}

		input := TaskControlRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		runAction, ok := mapTaskControlActionToRunControlAction(strings.TrimSpace(input.Action))
		if !ok {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "action must be one of cancel/retry/pause/resume", map[string]any{
				"action": input.Action,
			})
			return
		}

		forwardPayload, marshalErr := json.Marshal(map[string]string{"action": runAction})
		if marshalErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to build run control payload", map[string]any{})
			return
		}
		forwardRequest := r.Clone(r.Context())
		forwardRequest.Method = http.MethodPost
		forwardRequest.URL.Path = "/v1/runs/" + taskID + "/control"
		forwardRequest.SetPathValue("run_id", taskID)
		forwardRequest.Header = r.Header.Clone()
		forwardRequest.Header.Set("Content-Type", "application/json")
		forwardRequest.Body = io.NopCloser(bytes.NewReader(forwardPayload))
		forwardRequest.ContentLength = int64(len(forwardPayload))

		recorder := httptest.NewRecorder()
		RunControlHandler(state).ServeHTTP(recorder, forwardRequest)
		copyHTTPHeaders(w.Header(), recorder.Header())
		if recorder.Code != http.StatusOK {
			w.WriteHeader(recorder.Code)
			_, _ = w.Write(recorder.Body.Bytes())
			return
		}

		control := struct {
			OK            bool   `json:"ok"`
			RunID         string `json:"run_id"`
			State         string `json:"state"`
			PreviousState string `json:"previous_state"`
		}{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &control); err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to decode run control response", map[string]any{})
			return
		}
		writeJSON(w, http.StatusOK, TaskControlResponse{
			OK:            true,
			RunID:         runID,
			TaskID:        taskID,
			State:         control.State,
			PreviousState: control.PreviousState,
		})
	}
}

func buildRunTaskGraph(state *AppState, request *http.Request, runID string) (runtimeapplication.RunTaskGraph, bool) {
	if service, ok := newRunTaskQueryService(state); ok {
		graph, exists, err := service.BuildRunTaskGraph(request.Context(), runID)
		if err == nil {
			return graph, exists
		}
		log.Printf("runtime v1 run task graph query failed, fallback to in-memory map: %v", err)
	}
	return buildRunTaskGraphFromState(state, runID)
}

type runTaskGraphMetadata struct {
	MaxParallelism int
	ByExecutionID  map[string]runTaskExecutionMetadata
}

type runTaskExecutionMetadata struct {
	DependsOn  []string
	Priority   int
	RetryCount int
	MaxRetries int
	State      *string
	Artifact   *runtimeapplication.RunTaskArtifact
	LastError  *string
}

func buildRunTaskGraphFromState(state *AppState, runID string) (runtimeapplication.RunTaskGraph, bool) {
	if state == nil {
		return runtimeapplication.RunTaskGraph{}, false
	}
	state.mu.RLock()
	seedExecution, ok := state.executions[runID]
	if !ok {
		state.mu.RUnlock()
		return runtimeapplication.RunTaskGraph{}, false
	}
	metadata := deriveRunTaskGraphMetadata(state.executionEvents[seedExecution.ConversationID])
	inputs := make([]runtimeapplication.RunTaskInput, 0, len(state.executions))
	for _, execution := range state.executions {
		if execution.ConversationID != seedExecution.ConversationID {
			continue
		}
		taskMetadata := metadata.ByExecutionID[execution.ID]
		taskState := string(execution.State)
		if normalizedState, ok := normalizeTaskStateString(taskMetadata.State); ok {
			taskState = normalizedState
		}
		inputs = append(inputs, runtimeapplication.RunTaskInput{
			ExecutionID: execution.ID,
			State:       taskState,
			QueueIndex:  execution.QueueIndex,
			Priority:    taskMetadata.Priority,
			RetryCount:  taskMetadata.RetryCount,
			MaxRetries:  taskMetadata.MaxRetries,
			DependsOn:   append([]string{}, taskMetadata.DependsOn...),
			Artifact:    cloneRuntimeTaskArtifact(taskMetadata.Artifact),
			LastError:   cloneOptionalStringRunTask(taskMetadata.LastError),
			CreatedAt:   execution.CreatedAt,
			UpdatedAt:   execution.UpdatedAt,
		})
	}
	state.mu.RUnlock()
	if len(inputs) == 0 {
		return runtimeapplication.RunTaskGraph{}, false
	}
	return runtimeapplication.BuildRunTaskGraph(runID, metadata.MaxParallelism, inputs), true
}

func deriveRunTaskGraphMetadata(events []ExecutionEvent) runTaskGraphMetadata {
	metadata := runTaskGraphMetadata{
		MaxParallelism: 1,
		ByExecutionID:  map[string]runTaskExecutionMetadata{},
	}

	ordered := append([]ExecutionEvent{}, events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Sequence != ordered[j].Sequence {
			return ordered[i].Sequence < ordered[j].Sequence
		}
		if ordered[i].Timestamp != ordered[j].Timestamp {
			return ordered[i].Timestamp < ordered[j].Timestamp
		}
		return ordered[i].EventID < ordered[j].EventID
	})

	for _, event := range ordered {
		payload := event.Payload
		if payload == nil {
			payload = map[string]any{}
		}
		if maxParallelism, ok := parsePositiveInt(payload["max_parallelism"]); ok {
			metadata.MaxParallelism = maxParallelism
		}

		executionID := resolveTaskExecutionIDFromEvent(event, payload)
		if executionID == "" {
			continue
		}
		item := metadata.ByExecutionID[executionID]
		applyRunTaskExecutionMetadata(&item, payload, executionID)
		if nestedTask, ok := payload["task"].(map[string]any); ok {
			applyRunTaskExecutionMetadata(&item, nestedTask, executionID)
		}
		applyStructuredTaskEventMetadata(&item, event.Type, payload, executionID)
		if nestedTask, ok := payload["task"].(map[string]any); ok {
			applyStructuredTaskEventMetadata(&item, event.Type, nestedTask, executionID)
		}
		metadata.ByExecutionID[executionID] = item
	}
	return metadata
}

func applyRunTaskExecutionMetadata(item *runTaskExecutionMetadata, payload map[string]any, executionID string) {
	if item == nil {
		return
	}
	if state, ok := parseTaskState(payload["state"]); ok {
		item.State = &state
	}
	if dependsOn, ok := parseStringList(payload["depends_on"]); ok {
		item.DependsOn = dependsOn
	}
	if priority, ok := parseInt(payload["priority"]); ok {
		item.Priority = priority
	}
	if retryCount, ok := parseNonNegativeInt(payload["retry_count"]); ok {
		item.RetryCount = retryCount
	}
	if maxRetries, ok := parseNonNegativeInt(payload["max_retries"]); ok {
		item.MaxRetries = maxRetries
	}
	if artifact, ok := parseTaskArtifact(payload["task_artifact"], executionID); ok {
		item.Artifact = artifact
	}
	if artifact, ok := parseTaskArtifact(payload["artifact"], executionID); ok {
		item.Artifact = artifact
	}
	if message, ok := parseLastErrorMessage(payload); ok {
		item.LastError = &message
	}
}

func applyStructuredTaskEventMetadata(item *runTaskExecutionMetadata, eventType RunEventType, payload map[string]any, executionID string) {
	if item == nil {
		return
	}
	switch eventType {
	case RunEventTypeTaskDependenciesUpdated:
		if dependsOn, ok := parseStringList(payload["depends_on"]); ok {
			item.DependsOn = dependsOn
		}
		if priority, ok := parseInt(payload["priority"]); ok {
			item.Priority = priority
		}
	case RunEventTypeTaskRetryPolicyUpdated:
		if retryCount, ok := parseNonNegativeInt(payload["retry_count"]); ok {
			item.RetryCount = retryCount
		}
		if maxRetries, ok := parseNonNegativeInt(payload["max_retries"]); ok {
			item.MaxRetries = maxRetries
		}
	case RunEventTypeTaskArtifactEmitted:
		if artifact, ok := parseTaskArtifact(payload["artifact"], executionID); ok {
			item.Artifact = artifact
		} else if artifact, ok := parseTaskArtifact(payload["task_artifact"], executionID); ok {
			item.Artifact = artifact
		}
	case RunEventTypeTaskFailed:
		failedState := string(TaskStateFailed)
		item.State = &failedState
		if message := strings.TrimSpace(parseScalarString(payload["error_message"])); message != "" {
			item.LastError = &message
			return
		}
		if message, ok := parseLastErrorMessage(payload); ok {
			item.LastError = &message
		}
	case RunEventTypeTaskStarted:
		startedState := string(RunStateExecuting)
		item.State = &startedState
	case RunEventTypeTaskCompleted:
		completedState := string(TaskStateCompleted)
		item.State = &completedState
	case RunEventTypeTaskCancelled:
		cancelledState := string(TaskStateCancelled)
		item.State = &cancelledState
	}
}

func normalizeTaskStateString(state *string) (string, bool) {
	if state == nil {
		return "", false
	}
	normalized := strings.TrimSpace(*state)
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

func parseTaskState(value any) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(parseScalarString(value)))
	switch normalized {
	case string(TaskStateQueued):
		return string(TaskStateQueued), true
	case string(TaskStateBlocked):
		return string(TaskStateBlocked), true
	case string(TaskStateRunning):
		return string(RunStateExecuting), true
	case string(TaskStateRetrying):
		return string(TaskStateRetrying), true
	case string(TaskStateCompleted):
		return string(TaskStateCompleted), true
	case string(TaskStateFailed):
		return string(TaskStateFailed), true
	case string(TaskStateCancelled):
		return string(TaskStateCancelled), true
	case string(RunStatePending), string(RunStateExecuting), string(RunStateConfirming), string(RunStateAwaitingInput):
		return string(TaskStateRunning), true
	default:
		return "", false
	}
}

func parsePositiveInt(value any) (int, bool) {
	parsed, ok := parseInt(value)
	if !ok || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func parseNonNegativeInt(value any) (int, bool) {
	parsed, ok := parseInt(value)
	if !ok || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

func parseInt(value any) (int, bool) {
	switch raw := value.(type) {
	case int:
		return raw, true
	case int8:
		return int(raw), true
	case int16:
		return int(raw), true
	case int32:
		return int(raw), true
	case int64:
		return int(raw), true
	case uint:
		return int(raw), true
	case uint8:
		return int(raw), true
	case uint16:
		return int(raw), true
	case uint32:
		return int(raw), true
	case uint64:
		return int(raw), true
	case float32:
		return int(raw), true
	case float64:
		return int(raw), true
	case json.Number:
		if intValue, err := raw.Int64(); err == nil {
			return int(intValue), true
		}
		if floatValue, err := raw.Float64(); err == nil {
			return int(floatValue), true
		}
		return 0, false
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func parseStringList(value any) ([]string, bool) {
	switch raw := value.(type) {
	case []string:
		return uniqueTrimmedStrings(raw), true
	case []any:
		items := make([]string, 0, len(raw))
		for _, item := range raw {
			text, ok := item.(string)
			if !ok {
				continue
			}
			items = append(items, text)
		}
		return uniqueTrimmedStrings(items), true
	case string:
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return []string{}, true
		}
		if !strings.Contains(trimmed, ",") {
			return []string{trimmed}, true
		}
		return uniqueTrimmedStrings(strings.Split(trimmed, ",")), true
	default:
		return nil, false
	}
}

func uniqueTrimmedStrings(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := unique[trimmed]; exists {
			continue
		}
		unique[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}

func parseTaskArtifact(value any, executionID string) (*runtimeapplication.RunTaskArtifact, bool) {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}

	taskID := strings.TrimSpace(parseScalarString(raw["task_id"]))
	if taskID == "" {
		taskID = strings.TrimSpace(executionID)
	}
	kind := strings.TrimSpace(parseScalarString(raw["kind"]))
	uri := strings.TrimSpace(parseScalarString(raw["uri"]))
	summary := strings.TrimSpace(parseScalarString(raw["summary"]))

	metadata := map[string]any{}
	if rawMetadata, ok := raw["metadata"].(map[string]any); ok {
		metadata = cloneMapAny(rawMetadata)
	}
	if taskID == "" && kind == "" && uri == "" && summary == "" && len(metadata) == 0 {
		return nil, false
	}
	return &runtimeapplication.RunTaskArtifact{
		TaskID:   taskID,
		Kind:     kind,
		URI:      uri,
		Summary:  summary,
		Metadata: metadata,
	}, true
}

func parseLastErrorMessage(payload map[string]any) (string, bool) {
	if message := strings.TrimSpace(parseScalarString(payload["error_message"])); message != "" {
		return message, true
	}
	if message := strings.TrimSpace(parseScalarString(payload["last_error"])); message != "" {
		return message, true
	}
	switch rawError := payload["error"].(type) {
	case string:
		if message := strings.TrimSpace(rawError); message != "" {
			return message, true
		}
	case map[string]any:
		if message := strings.TrimSpace(parseScalarString(rawError["message"])); message != "" {
			return message, true
		}
		if message := strings.TrimSpace(parseScalarString(rawError["error"])); message != "" {
			return message, true
		}
	}
	if message := strings.TrimSpace(parseScalarString(payload["message"])); message != "" {
		return message, true
	}
	return "", false
}

func resolveTaskExecutionIDFromEvent(event ExecutionEvent, payload map[string]any) string {
	if taskID := strings.TrimSpace(parseScalarString(payload["task_id"])); taskID != "" {
		return taskID
	}
	if nestedTask, ok := payload["task"].(map[string]any); ok {
		if taskID := strings.TrimSpace(parseScalarString(nestedTask["task_id"])); taskID != "" {
			return taskID
		}
	}
	return strings.TrimSpace(event.ExecutionID)
}

func parseScalarString(value any) string {
	switch raw := value.(type) {
	case string:
		return raw
	case json.Number:
		return raw.String()
	case int:
		return strconv.Itoa(raw)
	case int8:
		return strconv.Itoa(int(raw))
	case int16:
		return strconv.Itoa(int(raw))
	case int32:
		return strconv.Itoa(int(raw))
	case int64:
		return strconv.FormatInt(raw, 10)
	case uint:
		return strconv.FormatUint(uint64(raw), 10)
	case uint8:
		return strconv.FormatUint(uint64(raw), 10)
	case uint16:
		return strconv.FormatUint(uint64(raw), 10)
	case uint32:
		return strconv.FormatUint(uint64(raw), 10)
	case uint64:
		return strconv.FormatUint(raw, 10)
	case float32:
		return strconv.FormatFloat(float64(raw), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(raw, 'f', -1, 64)
	default:
		return ""
	}
}

func cloneRuntimeTaskArtifact(input *runtimeapplication.RunTaskArtifact) *runtimeapplication.RunTaskArtifact {
	if input == nil {
		return nil
	}
	return &runtimeapplication.RunTaskArtifact{
		TaskID:   input.TaskID,
		Kind:     input.Kind,
		URI:      input.URI,
		Summary:  input.Summary,
		Metadata: cloneMapAny(input.Metadata),
	}
}

func cloneOptionalStringRunTask(input *string) *string {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func mapTaskControlActionToRunControlAction(action string) (string, bool) {
	switch strings.TrimSpace(action) {
	case "cancel", "pause":
		return "stop", true
	case "resume", "retry":
		return "resume", true
	default:
		return "", false
	}
}

func isValidTaskState(state TaskState) bool {
	switch state {
	case TaskStateQueued, TaskStateBlocked, TaskStateRunning, TaskStateRetrying, TaskStateCompleted, TaskStateFailed, TaskStateCancelled:
		return true
	default:
		return false
	}
}

func paginateTaskNodes(items []TaskNode, start int, limit int) ([]TaskNode, *string) {
	raw := make([]any, 0, len(items))
	for _, item := range items {
		raw = append(raw, item)
	}
	pagedRaw, next := paginateAny(raw, start, limit)
	paged := make([]TaskNode, 0, len(pagedRaw))
	for _, item := range pagedRaw {
		task, ok := item.(TaskNode)
		if !ok {
			continue
		}
		paged = append(paged, task)
	}
	return paged, next
}

func toHTTPAPIAgentGraph(input runtimeapplication.RunTaskGraph) AgentGraph {
	tasks := make([]TaskNode, 0, len(input.Tasks))
	for _, item := range input.Tasks {
		tasks = append(tasks, toHTTPAPITaskNode(item))
	}
	edges := make([]RunGraphEdge, 0, len(input.Edges))
	for _, item := range input.Edges {
		edges = append(edges, RunGraphEdge{
			FromTaskID: item.FromTaskID,
			ToTaskID:   item.ToTaskID,
		})
	}
	return AgentGraph{
		RunID:          input.RunID,
		MaxParallelism: input.MaxParallelism,
		Tasks:          tasks,
		Edges:          edges,
	}
}

func toHTTPAPITaskNode(input runtimeapplication.RunTaskNode) TaskNode {
	task := TaskNode{
		TaskID:      input.TaskID,
		RunID:       input.RunID,
		Title:       input.Title,
		Description: input.Description,
		State:       TaskState(input.State),
		AgentID:     input.AgentID,
		DependsOn:   append([]string{}, input.DependsOn...),
		Children:    append([]string{}, input.Children...),
		RetryCount:  input.RetryCount,
		MaxRetries:  input.MaxRetries,
		LastError:   input.LastError,
		CreatedAt:   input.CreatedAt,
		UpdatedAt:   input.UpdatedAt,
	}
	if input.Artifact != nil {
		task.Artifact = &TaskArtifact{
			TaskID:   input.Artifact.TaskID,
			Kind:     input.Artifact.Kind,
			URI:      input.Artifact.URI,
			Summary:  input.Artifact.Summary,
			Metadata: cloneMapAny(input.Artifact.Metadata),
		}
	}
	return task
}

func copyHTTPHeaders(dst http.Header, src http.Header) {
	for key := range dst {
		dst.Del(key)
	}
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
