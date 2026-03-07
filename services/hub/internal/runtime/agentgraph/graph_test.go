package agentgraph

import "testing"

func TestResolveTaskGraphUsesExplicitDependenciesAndDropsInvalid(t *testing.T) {
	graph := ResolveTaskGraph([]string{"task_a", "task_b", "task_c"}, map[string][]string{
		"task_a": {"missing"},
		"task_b": {"task_a", "task_a", " task_a "},
		"task_c": {"task_a", "task_c"},
	})

	if graph.UsedFallback {
		t.Fatalf("expected explicit graph without fallback, got %#v", graph)
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %#v", graph.Edges)
	}
	if graph.Edges[0].FromTaskID != "task_a" || graph.Edges[0].ToTaskID != "task_b" {
		t.Fatalf("unexpected first edge: %#v", graph.Edges[0])
	}
	if graph.Edges[1].FromTaskID != "task_a" || graph.Edges[1].ToTaskID != "task_c" {
		t.Fatalf("unexpected second edge: %#v", graph.Edges[1])
	}
	if len(graph.DependsOnByTask["task_a"]) != 0 {
		t.Fatalf("expected task_a without dependencies, got %#v", graph.DependsOnByTask["task_a"])
	}
}

func TestResolveTaskGraphFallsBackToSequentialOnCycle(t *testing.T) {
	graph := ResolveTaskGraph([]string{"task_a", "task_b", "task_c"}, map[string][]string{
		"task_a": {"task_c"},
		"task_b": {"task_a"},
		"task_c": {"task_b"},
	})

	if !graph.UsedFallback {
		t.Fatalf("expected cycle fallback, got %#v", graph)
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("expected sequential fallback edges, got %#v", graph.Edges)
	}
	if graph.Edges[0].FromTaskID != "task_a" || graph.Edges[0].ToTaskID != "task_b" {
		t.Fatalf("unexpected first fallback edge: %#v", graph.Edges[0])
	}
	if graph.Edges[1].FromTaskID != "task_b" || graph.Edges[1].ToTaskID != "task_c" {
		t.Fatalf("unexpected second fallback edge: %#v", graph.Edges[1])
	}
}
