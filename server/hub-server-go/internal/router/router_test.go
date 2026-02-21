package router

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/config"
)

func TestExecutionCommitAndPatchRoutesRegistered(t *testing.T) {
	cfg := &config.Config{
		TokenExpiryHours:  24,
		HubInternalSecret: "internal-secret",
	}

	handler := New(cfg, nil, nil, nil)
	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatalf("router does not implement chi.Routes")
	}

	registered := map[string]bool{}
	if err := chi.Walk(routes, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[fmt.Sprintf("%s %s", method, route)] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	if !registered["POST /v1/executions/{execution_id}/commit"] {
		t.Fatalf("missing route POST /v1/executions/{execution_id}/commit")
	}
	if !registered["GET /v1/executions/{execution_id}/patch"] {
		t.Fatalf("missing route GET /v1/executions/{execution_id}/patch")
	}
	if !registered["DELETE /v1/executions/{execution_id}/discard"] {
		t.Fatalf("missing route DELETE /v1/executions/{execution_id}/discard")
	}

	// Phase 5 routes
	for _, route := range []string{
		"GET /v1/projects",
		"POST /v1/projects",
		"DELETE /v1/projects/{project_id}",
		"POST /v1/projects/{project_id}/sync",
	} {
		if !registered[route] {
			t.Fatalf("missing route %s", route)
		}
	}

	// Phase 6 routes
	for _, route := range []string{
		"GET /v1/skill-sets",
		"POST /v1/skill-sets",
		"PUT /v1/skill-sets/{skill_set_id}",
		"DELETE /v1/skill-sets/{skill_set_id}",
		"GET /v1/skill-sets/{skill_set_id}/skills",
		"POST /v1/skill-sets/{skill_set_id}/skills",
		"DELETE /v1/skills/{skill_id}",
		"GET /v1/mcp-connectors",
		"POST /v1/mcp-connectors",
		"PUT /v1/mcp-connectors/{connector_id}",
		"DELETE /v1/mcp-connectors/{connector_id}",
	} {
		if !registered[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}
