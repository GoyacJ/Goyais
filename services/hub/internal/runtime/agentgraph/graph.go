package agentgraph

import (
	"sort"
	"strings"
)

type Edge struct {
	FromTaskID string
	ToTaskID   string
}

type TaskGraph struct {
	DependsOnByTask map[string][]string
	ChildrenByTask  map[string][]string
	Edges           []Edge
	UsedFallback    bool
}

func ResolveTaskGraph(taskOrder []string, explicitDependsOn map[string][]string) TaskGraph {
	order := make([]string, 0, len(taskOrder))
	indexByTask := make(map[string]int, len(taskOrder))
	for _, rawID := range taskOrder {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, exists := indexByTask[id]; exists {
			continue
		}
		indexByTask[id] = len(order)
		order = append(order, id)
	}

	depends := make(map[string][]string, len(order))
	for _, taskID := range order {
		depends[taskID] = []string{}
	}

	hasExplicitDependencies := false
	for _, taskID := range order {
		rawDeps := explicitDependsOn[taskID]
		if len(rawDeps) == 0 {
			continue
		}
		unique := make(map[string]struct{}, len(rawDeps))
		normalized := make([]string, 0, len(rawDeps))
		for _, rawDep := range rawDeps {
			depID := strings.TrimSpace(rawDep)
			if depID == "" || depID == taskID {
				continue
			}
			if _, exists := indexByTask[depID]; !exists {
				continue
			}
			if _, exists := unique[depID]; exists {
				continue
			}
			unique[depID] = struct{}{}
			normalized = append(normalized, depID)
		}
		sort.SliceStable(normalized, func(i, j int) bool {
			return indexByTask[normalized[i]] < indexByTask[normalized[j]]
		})
		if len(normalized) > 0 {
			hasExplicitDependencies = true
		}
		depends[taskID] = normalized
	}

	if !hasExplicitDependencies || hasCycle(order, depends) {
		return buildSequentialGraph(order)
	}
	return buildGraph(order, depends, false)
}

func buildSequentialGraph(order []string) TaskGraph {
	depends := make(map[string][]string, len(order))
	for idx, taskID := range order {
		if idx == 0 {
			depends[taskID] = []string{}
			continue
		}
		depends[taskID] = []string{order[idx-1]}
	}
	return buildGraph(order, depends, true)
}

func buildGraph(order []string, depends map[string][]string, usedFallback bool) TaskGraph {
	children := make(map[string][]string, len(order))
	for _, taskID := range order {
		children[taskID] = []string{}
	}

	edges := make([]Edge, 0, len(order))
	for _, taskID := range order {
		for _, depID := range depends[taskID] {
			children[depID] = append(children[depID], taskID)
			edges = append(edges, Edge{
				FromTaskID: depID,
				ToTaskID:   taskID,
			})
		}
	}

	return TaskGraph{
		DependsOnByTask: depends,
		ChildrenByTask:  children,
		Edges:           edges,
		UsedFallback:    usedFallback,
	}
}

func hasCycle(order []string, depends map[string][]string) bool {
	visited := make(map[string]uint8, len(order))
	var dfs func(taskID string) bool
	dfs = func(taskID string) bool {
		switch visited[taskID] {
		case 1:
			return true
		case 2:
			return false
		}
		visited[taskID] = 1
		for _, depID := range depends[taskID] {
			if dfs(depID) {
				return true
			}
		}
		visited[taskID] = 2
		return false
	}

	for _, taskID := range order {
		if dfs(taskID) {
			return true
		}
	}
	return false
}
