package scheduler

import (
	"sort"
	"strings"
)

const (
	TaskStateQueued    = "queued"
	TaskStateBlocked   = "blocked"
	TaskStateRunning   = "running"
	TaskStateRetrying  = "retrying"
	TaskStateCompleted = "completed"
	TaskStateFailed    = "failed"
	TaskStateCancelled = "cancelled"
)

const defaultPriority = 100

type Task struct {
	TaskID     string
	State      string
	DependsOn  []string
	Priority   int
	QueueIndex int
	RetryCount int
	MaxRetries int
	CreatedAt  string
}

func NormalizeTaskStates(tasks []Task, maxParallelism int) map[string]string {
	if maxParallelism <= 0 {
		maxParallelism = 1
	}

	ordered := sortedTasks(tasks)
	normalized := make(map[string]string, len(ordered))
	activeCount := 0
	for _, task := range ordered {
		id := strings.TrimSpace(task.TaskID)
		if id == "" {
			continue
		}
		state := normalizeState(task.State)
		normalized[id] = state
		if isActiveState(state) {
			activeCount++
		}
	}

	availableSlots := maxParallelism - activeCount
	if availableSlots < 0 {
		availableSlots = 0
	}

	for _, task := range ordered {
		id := strings.TrimSpace(task.TaskID)
		if id == "" {
			continue
		}
		state := normalized[id]
		if isStableTerminalState(state) {
			continue
		}
		dependenciesReady := areDependenciesCompleted(normalized, task.DependsOn)
		switch state {
		case TaskStateRunning:
			continue
		case TaskStateRetrying:
			if !dependenciesReady {
				normalized[id] = TaskStateBlocked
			}
		case TaskStateFailed:
			if task.RetryCount < task.MaxRetries {
				if dependenciesReady && availableSlots > 0 {
					normalized[id] = TaskStateRetrying
					availableSlots--
				} else {
					normalized[id] = TaskStateBlocked
				}
				continue
			}
		case TaskStateQueued, TaskStateBlocked:
			if !dependenciesReady {
				normalized[id] = TaskStateBlocked
				continue
			}
			if availableSlots > 0 {
				normalized[id] = TaskStateQueued
				availableSlots--
			} else {
				normalized[id] = TaskStateBlocked
			}
		default:
			if !dependenciesReady {
				normalized[id] = TaskStateBlocked
				continue
			}
			if availableSlots > 0 {
				normalized[id] = TaskStateQueued
				availableSlots--
			} else {
				normalized[id] = TaskStateBlocked
			}
		}
	}

	return normalized
}

func sortedTasks(tasks []Task) []Task {
	ordered := append([]Task{}, tasks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		pi := ordered[i].Priority
		pj := ordered[j].Priority
		if pi <= 0 {
			pi = defaultPriority
		}
		if pj <= 0 {
			pj = defaultPriority
		}
		if pi != pj {
			return pi < pj
		}
		if ordered[i].QueueIndex != ordered[j].QueueIndex {
			return ordered[i].QueueIndex < ordered[j].QueueIndex
		}
		if ordered[i].CreatedAt != ordered[j].CreatedAt {
			return ordered[i].CreatedAt < ordered[j].CreatedAt
		}
		return strings.TrimSpace(ordered[i].TaskID) < strings.TrimSpace(ordered[j].TaskID)
	})
	return ordered
}

func areDependenciesCompleted(states map[string]string, dependencies []string) bool {
	for _, dep := range dependencies {
		depID := strings.TrimSpace(dep)
		if depID == "" {
			continue
		}
		if states[depID] != TaskStateCompleted {
			return false
		}
	}
	return true
}

func normalizeState(raw string) string {
	switch strings.TrimSpace(raw) {
	case TaskStateQueued:
		return TaskStateQueued
	case TaskStateBlocked:
		return TaskStateBlocked
	case TaskStateRunning:
		return TaskStateRunning
	case TaskStateRetrying:
		return TaskStateRetrying
	case TaskStateCompleted:
		return TaskStateCompleted
	case TaskStateFailed:
		return TaskStateFailed
	case TaskStateCancelled:
		return TaskStateCancelled
	default:
		return TaskStateBlocked
	}
}

func isActiveState(state string) bool {
	return state == TaskStateRunning || state == TaskStateRetrying
}

func isStableTerminalState(state string) bool {
	return state == TaskStateCompleted || state == TaskStateCancelled
}
