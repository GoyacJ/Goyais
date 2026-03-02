package scheduler

import "testing"

func TestNormalizeTaskStatesRespectsPriorityAndParallelism(t *testing.T) {
	states := NormalizeTaskStates([]Task{
		{
			TaskID:     "task_low_priority",
			State:      "queued",
			Priority:   100,
			QueueIndex: 1,
			DependsOn:  []string{},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:01Z",
		},
		{
			TaskID:     "task_high_priority",
			State:      "queued",
			Priority:   10,
			QueueIndex: 2,
			DependsOn:  []string{},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:02Z",
		},
	}, 1)

	if states["task_high_priority"] != "queued" {
		t.Fatalf("expected high priority task queued, got %q", states["task_high_priority"])
	}
	if states["task_low_priority"] != "blocked" {
		t.Fatalf("expected low priority task blocked by parallelism, got %q", states["task_low_priority"])
	}
}

func TestNormalizeTaskStatesAppliesDependencyAndRetryBudget(t *testing.T) {
	states := NormalizeTaskStates([]Task{
		{
			TaskID:     "task_root",
			State:      "completed",
			Priority:   100,
			QueueIndex: 0,
			DependsOn:  []string{},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:00Z",
		},
		{
			TaskID:     "task_retry",
			State:      "failed",
			Priority:   100,
			QueueIndex: 1,
			DependsOn:  []string{"task_root"},
			RetryCount: 0,
			MaxRetries: 2,
			CreatedAt:  "2026-03-01T00:00:01Z",
		},
		{
			TaskID:     "task_leaf",
			State:      "queued",
			Priority:   100,
			QueueIndex: 2,
			DependsOn:  []string{"task_retry"},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:02Z",
		},
	}, 1)

	if states["task_retry"] != "retrying" {
		t.Fatalf("expected retry task in retrying state, got %q", states["task_retry"])
	}
	if states["task_leaf"] != "blocked" {
		t.Fatalf("expected leaf blocked until retry task completed, got %q", states["task_leaf"])
	}
}

func TestNormalizeTaskStatesRunningTaskConsumesSlot(t *testing.T) {
	states := NormalizeTaskStates([]Task{
		{
			TaskID:     "task_running",
			State:      "running",
			Priority:   100,
			QueueIndex: 0,
			DependsOn:  []string{},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:00Z",
		},
		{
			TaskID:     "task_waiting",
			State:      "queued",
			Priority:   100,
			QueueIndex: 1,
			DependsOn:  []string{},
			RetryCount: 0,
			MaxRetries: 0,
			CreatedAt:  "2026-03-01T00:00:01Z",
		},
	}, 1)

	if states["task_running"] != "running" {
		t.Fatalf("expected running task to stay running, got %q", states["task_running"])
	}
	if states["task_waiting"] != "blocked" {
		t.Fatalf("expected waiting task blocked when no slots available, got %q", states["task_waiting"])
	}
}
