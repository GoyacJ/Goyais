package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type workspaceProjectConfigItem struct {
	ProjectID   string        `json:"project_id"`
	ProjectName string        `json:"project_name"`
	Config      ProjectConfig `json:"config"`
}

func CatalogRootHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.read",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			root, err := state.GetCatalogRoot(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "CATALOG_ROOT_READ_FAILED", "Failed to read catalog root", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusOK, root)
		case http.MethodPut:
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"catalog.update_root",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin, RoleApprover, RoleDeveloper,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			workspace, exists := state.GetWorkspace(workspaceID)
			if exists && workspace.Mode == WorkspaceModeRemote && session.Role != RoleAdmin {
				WriteStandardError(w, r, http.StatusForbidden, "ACCESS_DENIED", "Only admin can update remote catalog root", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			input := CatalogRootUpdateRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			root, err := state.SetCatalogRoot(workspaceID, input.CatalogRoot)
			if err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "catalog_root is invalid", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			if state.authz != nil {
				_ = state.authz.appendAudit(workspaceID, session.UserID, "catalog.update_root", "workspace", workspaceID, "success", map[string]any{
					"catalog_root": root.CatalogRoot,
				}, TraceIDFromContext(r.Context()))
			}
			writeJSON(w, http.StatusOK, root)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func ResourceConfigsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.read",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			query := resourceConfigQuery{
				Type:  ResourceType(strings.TrimSpace(r.URL.Query().Get("type"))),
				Query: strings.TrimSpace(r.URL.Query().Get("q")),
			}
			if rawEnabled := strings.TrimSpace(r.URL.Query().Get("enabled")); rawEnabled != "" {
				enabled := rawEnabled == "true" || rawEnabled == "1"
				query.Enabled = &enabled
			}
			items, err := listWorkspaceResourceConfigs(state, workspaceID, query)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_CONFIG_LIST_FAILED", "Failed to list resource configs", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.write",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin, RoleApprover, RoleDeveloper,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			input := ResourceConfigCreateRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			if err := validateCreateResourceConfig(input); err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
				return
			}
			enabled := true
			if input.Enabled != nil {
				enabled = *input.Enabled
			}
			now := nowUTC()
			config := ResourceConfig{
				ID:          "rc_" + randomHex(6),
				WorkspaceID: workspaceID,
				Type:        input.Type,
				Name:        strings.TrimSpace(input.Name),
				Enabled:     enabled,
				Model:       input.Model,
				Rule:        input.Rule,
				Skill:       input.Skill,
				MCP:         input.MCP,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			created, err := saveWorkspaceResourceConfig(state, config)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_CONFIG_CREATE_FAILED", "Failed to create resource config", map[string]any{})
				return
			}
			if state.authz != nil {
				_ = state.authz.appendAudit(workspaceID, session.UserID, "resource_config.write", "resource_config", created.ID, "success", map[string]any{
					"operation": "create",
					"type":      created.Type,
				}, TraceIDFromContext(r.Context()))
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func ResourceConfigByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		configID := strings.TrimSpace(r.PathValue("config_id"))
		switch r.Method {
		case http.MethodPatch:
			_, authErr := authorizeAction(state, r, workspaceID, "resource_config.write", authorizationResource{WorkspaceID: workspaceID}, authorizationContext{OperationType: "write", ABACRequired: true}, RoleAdmin, RoleApprover, RoleDeveloper)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			origin, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, configID)
			if err != nil || !exists {
				WriteStandardError(w, r, http.StatusNotFound, "RESOURCE_CONFIG_NOT_FOUND", "Resource config does not exist", map[string]any{"config_id": configID})
				return
			}
			patch := ResourceConfigPatchRequest{}
			if err := decodeJSONBody(r, &patch); err != nil {
				err.write(w, r)
				return
			}
			applyPatchToResourceConfig(&origin, patch)
			origin.UpdatedAt = nowUTC()
			updated, err := saveWorkspaceResourceConfig(state, origin)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_CONFIG_UPDATE_FAILED", "Failed to update resource config", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			_, authErr := authorizeAction(state, r, workspaceID, "resource_config.delete", authorizationResource{WorkspaceID: workspaceID}, authorizationContext{OperationType: "write", ABACRequired: true}, RoleAdmin, RoleApprover)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if err := removeWorkspaceResourceConfig(state, workspaceID, configID); err != nil {
				WriteStandardError(w, r, http.StatusNotFound, "RESOURCE_CONFIG_NOT_FOUND", "Resource config does not exist", map[string]any{"config_id": configID})
				return
			}
			writeJSON(w, http.StatusNoContent, map[string]any{})
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func WorkspaceProjectConfigsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		_, authErr := authorizeAction(state, r, workspaceID, "project_config.read", authorizationResource{WorkspaceID: workspaceID}, authorizationContext{OperationType: "read"})
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		state.mu.RLock()
		items := make([]workspaceProjectConfigItem, 0)
		for _, project := range state.projects {
			if project.WorkspaceID != workspaceID {
				continue
			}
			config := state.projectConfigs[project.ID]
			items = append(items, workspaceProjectConfigItem{
				ProjectID:   project.ID,
				ProjectName: project.Name,
				Config:      config,
			})
		}
		state.mu.RUnlock()
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].ProjectName) < strings.ToLower(items[j].ProjectName)
		})
		writeJSON(w, http.StatusOK, items)
	}
}

func validateCreateResourceConfig(input ResourceConfigCreateRequest) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("name is required")
	}
	switch input.Type {
	case ResourceTypeModel:
		if input.Model == nil || strings.TrimSpace(input.Model.ModelID) == "" {
			return errors.New("model spec with model_id is required")
		}
	case ResourceTypeRule:
		if input.Rule == nil || strings.TrimSpace(input.Rule.Content) == "" {
			return errors.New("rule spec with content is required")
		}
	case ResourceTypeSkill:
		if input.Skill == nil || strings.TrimSpace(input.Skill.Content) == "" {
			return errors.New("skill spec with content is required")
		}
	case ResourceTypeMCP:
		if input.MCP == nil || strings.TrimSpace(input.MCP.Transport) == "" {
			return errors.New("mcp spec with transport is required")
		}
	default:
		return errors.New("resource type is invalid")
	}
	return nil
}

func applyPatchToResourceConfig(target *ResourceConfig, patch ResourceConfigPatchRequest) {
	if patch.Name != nil {
		target.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Enabled != nil {
		target.Enabled = *patch.Enabled
	}
	if patch.Model != nil {
		nextModel := *patch.Model
		// Keep existing API key when client edits other fields without re-sending secret.
		if strings.TrimSpace(nextModel.APIKey) == "" && target.Model != nil {
			nextModel.APIKey = target.Model.APIKey
		}
		target.Model = &nextModel
	}
	if patch.Rule != nil {
		target.Rule = patch.Rule
	}
	if patch.Skill != nil {
		target.Skill = patch.Skill
	}
	if patch.MCP != nil {
		target.MCP = patch.MCP
	}
}

func listWorkspaceResourceConfigs(state *AppState, workspaceID string, query resourceConfigQuery) ([]ResourceConfig, error) {
	if state.authz != nil {
		return state.authz.listResourceConfigs(workspaceID, query)
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	items := make([]ResourceConfig, 0)
	for _, item := range state.resourceConfigs {
		if item.WorkspaceID != workspaceID {
			continue
		}
		if query.Type != "" && item.Type != query.Type {
			continue
		}
		if query.Enabled != nil && item.Enabled != *query.Enabled {
			continue
		}
		if query.Query != "" && !strings.Contains(strings.ToLower(item.Name), strings.ToLower(query.Query)) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
	return items, nil
}

func saveWorkspaceResourceConfig(state *AppState, item ResourceConfig) (ResourceConfig, error) {
	if state.authz != nil {
		return state.authz.upsertResourceConfig(item)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.resourceConfigs[item.ID] = item
	return item, nil
}

func loadWorkspaceResourceConfigRaw(state *AppState, workspaceID string, configID string) (ResourceConfig, bool, error) {
	if state.authz != nil {
		return state.authz.getResourceConfigRaw(workspaceID, configID)
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	item, ok := state.resourceConfigs[configID]
	if !ok || item.WorkspaceID != workspaceID {
		return ResourceConfig{}, false, nil
	}
	return item, true, nil
}

func removeWorkspaceResourceConfig(state *AppState, workspaceID string, configID string) error {
	if state.authz != nil {
		return state.authz.deleteResourceConfig(workspaceID, configID)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	item, ok := state.resourceConfigs[configID]
	if !ok || item.WorkspaceID != workspaceID {
		return sql.ErrNoRows
	}
	delete(state.resourceConfigs, configID)
	return nil
}

func isValidURLString(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.TrimSpace(parsed.Host) != ""
}
