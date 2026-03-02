package application

import (
	"sort"
	"strings"

	runtimeagentgraph "goyais/services/hub/internal/runtime/agentgraph"
	runtimescheduler "goyais/services/hub/internal/runtime/scheduler"
)

const (
	RunTaskStateQueued    = "queued"
	RunTaskStateBlocked   = "blocked"
	RunTaskStateRunning   = "running"
	RunTaskStateRetrying  = "retrying"
	RunTaskStateCompleted = "completed"
	RunTaskStateFailed    = "failed"
	RunTaskStateCancelled = "cancelled"
)

type RunTaskInput struct {
	ExecutionID string
	State       string
	QueueIndex  int
	Priority    int
	RetryCount  int
	MaxRetries  int
	DependsOn   []string
	Artifact    *RunTaskArtifact
	LastError   *string
	CreatedAt   string
	UpdatedAt   string
}

type RunTaskArtifact struct {
	TaskID   string
	Kind     string
	URI      string
	Summary  string
	Metadata map[string]any
}

type RunTaskNode struct {
	TaskID      string
	RunID       string
	Title       string
	Description string
	State       string
	AgentID     string
	DependsOn   []string
	Children    []string
	RetryCount  int
	MaxRetries  int
	Artifact    *RunTaskArtifact
	LastError   *string
	CreatedAt   string
	UpdatedAt   string
}

type RunGraphEdge struct {
	FromTaskID string
	ToTaskID   string
}

type RunTaskGraph struct {
	RunID          string
	MaxParallelism int
	Tasks          []RunTaskNode
	Edges          []RunGraphEdge
}

func BuildRunTaskGraph(runID string, maxParallelism int, inputs []RunTaskInput) RunTaskGraph {
	if maxParallelism <= 0 {
		maxParallelism = 1
	}

	ordered := append([]RunTaskInput{}, inputs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].QueueIndex == ordered[j].QueueIndex {
			if ordered[i].CreatedAt == ordered[j].CreatedAt {
				return ordered[i].ExecutionID < ordered[j].ExecutionID
			}
			return ordered[i].CreatedAt < ordered[j].CreatedAt
		}
		return ordered[i].QueueIndex < ordered[j].QueueIndex
	})

	tasks := make([]RunTaskNode, 0, len(ordered))
	seenTaskIDs := make(map[string]struct{}, len(ordered))
	priorityByTaskID := make(map[string]int, len(ordered))
	dependsByTaskID := make(map[string][]string, len(ordered))
	for _, input := range ordered {
		id := strings.TrimSpace(input.ExecutionID)
		if id == "" {
			continue
		}
		if _, exists := seenTaskIDs[id]; exists {
			continue
		}
		seenTaskIDs[id] = struct{}{}
		retryCount := input.RetryCount
		if retryCount < 0 {
			retryCount = 0
		}
		maxRetries := input.MaxRetries
		if maxRetries < 0 {
			maxRetries = 0
		}
		priorityByTaskID[id] = input.Priority
		dependsByTaskID[id] = append([]string{}, input.DependsOn...)
		artifact := cloneRunTaskArtifact(input.Artifact)
		lastError := cloneOptionalString(input.LastError)
		tasks = append(tasks, RunTaskNode{
			TaskID:      id,
			RunID:       runID,
			Title:       "Execution " + id,
			State:       mapExecutionStateToTaskState(input.State),
			DependsOn:   []string{},
			Children:    []string{},
			RetryCount:  retryCount,
			MaxRetries:  maxRetries,
			Artifact:    artifact,
			LastError:   lastError,
			CreatedAt:   input.CreatedAt,
			UpdatedAt:   input.UpdatedAt,
			Description: "",
			AgentID:     "",
		})
	}

	taskOrder := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskOrder = append(taskOrder, task.TaskID)
	}
	resolvedGraph := runtimeagentgraph.ResolveTaskGraph(taskOrder, dependsByTaskID)
	for idx := range tasks {
		tasks[idx].DependsOn = append([]string{}, resolvedGraph.DependsOnByTask[tasks[idx].TaskID]...)
		tasks[idx].Children = append([]string{}, resolvedGraph.ChildrenByTask[tasks[idx].TaskID]...)
	}
	edges := make([]RunGraphEdge, 0, len(resolvedGraph.Edges))
	for _, edge := range resolvedGraph.Edges {
		edges = append(edges, RunGraphEdge{
			FromTaskID: edge.FromTaskID,
			ToTaskID:   edge.ToTaskID,
		})
	}
	schedulerTasks := make([]runtimescheduler.Task, 0, len(tasks))
	for idx, task := range tasks {
		schedulerTasks = append(schedulerTasks, runtimescheduler.Task{
			TaskID:     task.TaskID,
			State:      task.State,
			DependsOn:  append([]string{}, task.DependsOn...),
			Priority:   priorityByTaskID[task.TaskID],
			QueueIndex: idx,
			RetryCount: task.RetryCount,
			MaxRetries: task.MaxRetries,
			CreatedAt:  task.CreatedAt,
		})
	}
	normalizedStates := runtimescheduler.NormalizeTaskStates(schedulerTasks, maxParallelism)
	for idx := range tasks {
		state, ok := normalizedStates[tasks[idx].TaskID]
		if !ok {
			continue
		}
		tasks[idx].State = state
	}

	return RunTaskGraph{
		RunID:          runID,
		MaxParallelism: maxParallelism,
		Tasks:          tasks,
		Edges:          edges,
	}
}

func FilterRunTasksByState(tasks []RunTaskNode, state string) []RunTaskNode {
	normalized := strings.TrimSpace(state)
	if normalized == "" {
		return append([]RunTaskNode{}, tasks...)
	}
	filtered := make([]RunTaskNode, 0, len(tasks))
	for _, task := range tasks {
		if task.State == normalized {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func FindRunTaskByID(tasks []RunTaskNode, taskID string) (RunTaskNode, bool) {
	target := strings.TrimSpace(taskID)
	if target == "" {
		return RunTaskNode{}, false
	}
	for _, task := range tasks {
		if task.TaskID == target {
			return task, true
		}
	}
	return RunTaskNode{}, false
}

func mapExecutionStateToTaskState(raw string) string {
	switch strings.TrimSpace(raw) {
	case "queued", "pending":
		return RunTaskStateQueued
	case "executing":
		return RunTaskStateRunning
	case "retrying":
		return RunTaskStateRetrying
	case "confirming", "awaiting_input":
		return RunTaskStateBlocked
	case "completed":
		return RunTaskStateCompleted
	case "failed":
		return RunTaskStateFailed
	case "cancelled":
		return RunTaskStateCancelled
	default:
		return RunTaskStateBlocked
	}
}

func cloneRunTaskArtifact(input *RunTaskArtifact) *RunTaskArtifact {
	if input == nil {
		return nil
	}
	return &RunTaskArtifact{
		TaskID:   input.TaskID,
		Kind:     input.Kind,
		URI:      input.URI,
		Summary:  input.Summary,
		Metadata: cloneMapAny(input.Metadata),
	}
}

func cloneOptionalString(input *string) *string {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
