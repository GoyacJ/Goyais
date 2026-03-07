package application

import "testing"

func TestBuildRunTaskGraphMapsExecutionStatesAndSorts(t *testing.T) {
	graph := BuildRunTaskGraph("run_1", 0, []RunTaskInput{
		{
			ExecutionID: "exec_2",
			State:       " completed ",
			QueueIndex:  2,
			CreatedAt:   "2026-03-01T00:00:02Z",
			UpdatedAt:   "2026-03-01T00:00:02Z",
		},
		{
			ExecutionID: "exec_1",
			State:       " pending ",
			QueueIndex:  1,
			CreatedAt:   "2026-03-01T00:00:01Z",
			UpdatedAt:   "2026-03-01T00:00:01Z",
		},
		{
			ExecutionID: "exec_3",
			State:       " awaiting_input ",
			QueueIndex:  3,
			CreatedAt:   "2026-03-01T00:00:03Z",
			UpdatedAt:   "2026-03-01T00:00:03Z",
		},
	})

	if graph.RunID != "run_1" {
		t.Fatalf("expected run_id run_1, got %q", graph.RunID)
	}
	if graph.MaxParallelism != 1 {
		t.Fatalf("expected default max_parallelism 1, got %d", graph.MaxParallelism)
	}
	if len(graph.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %#v", graph.Tasks)
	}
	if graph.Tasks[0].TaskID != "exec_1" || graph.Tasks[0].State != RunTaskStateQueued {
		t.Fatalf("expected first task queued exec_1, got %#v", graph.Tasks[0])
	}
	if graph.Tasks[1].TaskID != "exec_2" || graph.Tasks[1].State != RunTaskStateCompleted {
		t.Fatalf("expected second task completed exec_2, got %#v", graph.Tasks[1])
	}
	if graph.Tasks[2].TaskID != "exec_3" || graph.Tasks[2].State != RunTaskStateBlocked {
		t.Fatalf("expected third task blocked exec_3, got %#v", graph.Tasks[2])
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 graph edges, got %#v", graph.Edges)
	}
	if graph.Edges[0].FromTaskID != "exec_1" || graph.Edges[0].ToTaskID != "exec_2" {
		t.Fatalf("unexpected first graph edge: %#v", graph.Edges[0])
	}
	if graph.Edges[1].FromTaskID != "exec_2" || graph.Edges[1].ToTaskID != "exec_3" {
		t.Fatalf("unexpected second graph edge: %#v", graph.Edges[1])
	}
	if len(graph.Tasks[1].DependsOn) != 1 || graph.Tasks[1].DependsOn[0] != "exec_1" {
		t.Fatalf("expected exec_2 depends_on exec_1, got %#v", graph.Tasks[1].DependsOn)
	}
	if len(graph.Tasks[1].Children) != 1 || graph.Tasks[1].Children[0] != "exec_3" {
		t.Fatalf("expected exec_2 child exec_3, got %#v", graph.Tasks[1].Children)
	}
}

func TestFilterAndFindRunTasks(t *testing.T) {
	tasks := []RunTaskNode{
		{TaskID: "exec_1", State: RunTaskStateQueued},
		{TaskID: "exec_2", State: RunTaskStateRunning},
		{TaskID: "exec_3", State: RunTaskStateRunning},
	}

	filtered := FilterRunTasksByState(tasks, "running")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 running tasks, got %#v", filtered)
	}
	found, ok := FindRunTaskByID(tasks, "exec_2")
	if !ok || found.TaskID != "exec_2" {
		t.Fatalf("expected find task exec_2, got ok=%v task=%#v", ok, found)
	}
	_, ok = FindRunTaskByID(tasks, "missing")
	if ok {
		t.Fatalf("expected missing task lookup false")
	}
}

func TestBuildRunTaskGraphAppliesRetryBudgetAndDependencyConstraints(t *testing.T) {
	graph := BuildRunTaskGraph("run_2", 1, []RunTaskInput{
		{
			ExecutionID: "exec_root",
			State:       "completed",
			QueueIndex:  0,
			CreatedAt:   "2026-03-01T00:00:00Z",
			UpdatedAt:   "2026-03-01T00:00:00Z",
		},
		{
			ExecutionID: "exec_retry",
			State:       "failed",
			QueueIndex:  1,
			RetryCount:  0,
			MaxRetries:  2,
			CreatedAt:   "2026-03-01T00:00:01Z",
			UpdatedAt:   "2026-03-01T00:00:01Z",
		},
		{
			ExecutionID: "exec_leaf",
			State:       "queued",
			QueueIndex:  2,
			CreatedAt:   "2026-03-01T00:00:02Z",
			UpdatedAt:   "2026-03-01T00:00:02Z",
		},
	})

	if len(graph.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %#v", graph.Tasks)
	}
	if graph.Tasks[1].State != RunTaskStateRetrying {
		t.Fatalf("expected retry task in retrying state, got %#v", graph.Tasks[1])
	}
	if graph.Tasks[2].State != RunTaskStateBlocked {
		t.Fatalf("expected leaf task blocked by dependency, got %#v", graph.Tasks[2])
	}
}

func TestBuildRunTaskGraphUsesExplicitDependencies(t *testing.T) {
	graph := BuildRunTaskGraph("run_3", 2, []RunTaskInput{
		{
			ExecutionID: "exec_a",
			State:       "queued",
			QueueIndex:  1,
			CreatedAt:   "2026-03-01T00:00:01Z",
			UpdatedAt:   "2026-03-01T00:00:01Z",
		},
		{
			ExecutionID: "exec_b",
			State:       "queued",
			QueueIndex:  2,
			DependsOn:   []string{"exec_a"},
			CreatedAt:   "2026-03-01T00:00:02Z",
			UpdatedAt:   "2026-03-01T00:00:02Z",
		},
		{
			ExecutionID: "exec_c",
			State:       "queued",
			QueueIndex:  3,
			DependsOn:   []string{"exec_a"},
			CreatedAt:   "2026-03-01T00:00:03Z",
			UpdatedAt:   "2026-03-01T00:00:03Z",
		},
	})

	if len(graph.Edges) != 2 {
		t.Fatalf("expected explicit dependency graph with 2 edges, got %#v", graph.Edges)
	}
	if graph.Edges[0].FromTaskID != "exec_a" || graph.Edges[0].ToTaskID != "exec_b" {
		t.Fatalf("unexpected first edge: %#v", graph.Edges[0])
	}
	if graph.Edges[1].FromTaskID != "exec_a" || graph.Edges[1].ToTaskID != "exec_c" {
		t.Fatalf("unexpected second edge: %#v", graph.Edges[1])
	}
	if len(graph.Tasks[0].Children) != 2 {
		t.Fatalf("expected root task with 2 children, got %#v", graph.Tasks[0].Children)
	}
	if len(graph.Tasks[1].DependsOn) != 1 || graph.Tasks[1].DependsOn[0] != "exec_a" {
		t.Fatalf("expected exec_b depends on exec_a, got %#v", graph.Tasks[1].DependsOn)
	}
	if len(graph.Tasks[2].DependsOn) != 1 || graph.Tasks[2].DependsOn[0] != "exec_a" {
		t.Fatalf("expected exec_c depends on exec_a, got %#v", graph.Tasks[2].DependsOn)
	}
}

func TestBuildRunTaskGraphCarriesArtifactAndLastError(t *testing.T) {
	lastError := "tool call failed"
	graph := BuildRunTaskGraph("run_4", 1, []RunTaskInput{
		{
			ExecutionID: "exec_a",
			State:       "completed",
			QueueIndex:  0,
			Artifact: &RunTaskArtifact{
				TaskID:  "exec_a",
				Kind:    "diff",
				URI:     "file:///tmp/a.patch",
				Summary: "patch generated",
				Metadata: map[string]any{
					"files": 2,
				},
			},
			LastError: &lastError,
			CreatedAt: "2026-03-01T00:00:00Z",
			UpdatedAt: "2026-03-01T00:00:01Z",
		},
	})

	if len(graph.Tasks) != 1 {
		t.Fatalf("expected one task, got %#v", graph.Tasks)
	}
	task := graph.Tasks[0]
	if task.Artifact == nil || task.Artifact.Kind != "diff" || task.Artifact.URI != "file:///tmp/a.patch" {
		t.Fatalf("expected artifact propagated, got %#v", task.Artifact)
	}
	if task.LastError == nil || *task.LastError != lastError {
		t.Fatalf("expected last error propagated, got %#v", task.LastError)
	}
}
