package httpapi

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

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
			if input.Model != nil {
				input.Model = normalizeModelSpecForStorage(input.Model)
				if err := validateModelSpecAgainstCatalog(state, workspaceID, nil, input.Model); err != nil {
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
					return
				}
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
			normalizeResourceConfigForStorage(&config)
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
			if patch.Model != nil {
				patch.Model = normalizeModelSpecForStorage(patch.Model)
			}
			currentModel := origin.Model
			applyPatchToResourceConfig(&origin, patch)
			if origin.Type == ResourceTypeModel && origin.Model != nil {
				if err := validateModelSpecAgainstCatalog(state, workspaceID, currentModel, origin.Model); err != nil {
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
					return
				}
			}
			normalizeResourceConfigForStorage(&origin)
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

func validateCreateResourceConfig(input ResourceConfigCreateRequest) error {
	switch input.Type {
	case ResourceTypeModel:
		if input.Model == nil || strings.TrimSpace(input.Model.ModelID) == "" {
			return errors.New("model spec with model_id is required")
		}
	case ResourceTypeRule:
		if strings.TrimSpace(input.Name) == "" {
			return errors.New("name is required")
		}
		if input.Rule == nil || strings.TrimSpace(input.Rule.Content) == "" {
			return errors.New("rule spec with content is required")
		}
	case ResourceTypeSkill:
		if strings.TrimSpace(input.Name) == "" {
			return errors.New("name is required")
		}
		if input.Skill == nil || strings.TrimSpace(input.Skill.Content) == "" {
			return errors.New("skill spec with content is required")
		}
	case ResourceTypeMCP:
		if strings.TrimSpace(input.Name) == "" {
			return errors.New("name is required")
		}
		if input.MCP == nil || strings.TrimSpace(input.MCP.Transport) == "" {
			return errors.New("mcp spec with transport is required")
		}
	default:
		return errors.New("resource type is invalid")
	}
	return nil
}

func applyPatchToResourceConfig(target *ResourceConfig, patch ResourceConfigPatchRequest) {
	if patch.Name != nil && target.Type != ResourceTypeModel {
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
		if query.Query != "" && !matchesResourceConfigQuery(item, query.Query) {
			continue
		}
		normalizeResourceConfigForStorage(&item)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
	return items, nil
}

func saveWorkspaceResourceConfig(state *AppState, item ResourceConfig) (ResourceConfig, error) {
	normalizeResourceConfigForStorage(&item)
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

func matchesResourceConfigQuery(item ResourceConfig, query string) bool {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery == "" {
		return true
	}
	return strings.Contains(strings.ToLower(resourceConfigSearchText(item)), lowerQuery)
}

func resourceConfigSearchText(item ResourceConfig) string {
	if item.Type == ResourceTypeModel && item.Model != nil {
		return strings.TrimSpace(string(item.Model.Vendor) + " " + item.Model.ModelID)
	}
	return strings.TrimSpace(item.Name)
}

func normalizeModelSpecForStorage(spec *ModelSpec) *ModelSpec {
	if spec == nil {
		return nil
	}
	next := *spec
	next.Vendor = ModelVendorName(strings.TrimSpace(string(next.Vendor)))
	next.ModelID = strings.TrimSpace(next.ModelID)
	next.BaseURL = strings.TrimSpace(next.BaseURL)
	next.BaseURLKey = strings.TrimSpace(next.BaseURLKey)
	if next.Vendor != ModelVendorLocal {
		next.BaseURL = ""
	}
	return &next
}

func normalizeResourceConfigForStorage(item *ResourceConfig) {
	if item == nil {
		return
	}
	if item.Type == ResourceTypeModel {
		item.Name = ""
	}
	if item.Model != nil {
		item.Model = normalizeModelSpecForStorage(item.Model)
	}
}

func validateModelSpecAgainstCatalog(state *AppState, workspaceID string, current *ModelSpec, next *ModelSpec) error {
	if next == nil {
		return errors.New("model spec with model_id is required")
	}
	vendor := ModelVendorName(strings.TrimSpace(string(next.Vendor)))
	modelID := strings.TrimSpace(next.ModelID)
	if vendor == "" || modelID == "" {
		return errors.New("model vendor and model_id are required")
	}
	response, err := state.LoadModelCatalog(workspaceID, false)
	if err != nil {
		return fmt.Errorf("failed to load model catalog: %w", err)
	}

	for _, item := range response.Vendors {
		if item.Name != vendor {
			continue
		}
		endpointKey := strings.TrimSpace(next.BaseURLKey)
		if endpointKey != "" {
			if _, ok := item.BaseURLs[endpointKey]; !ok {
				return fmt.Errorf("vendor %s does not contain endpoint key %s", vendor, endpointKey)
			}
		}
		for _, model := range item.Models {
			if strings.TrimSpace(model.ID) != modelID {
				continue
			}
			if model.Enabled {
				return nil
			}
			if current != nil && strings.TrimSpace(current.ModelID) == modelID && strings.TrimSpace(string(current.Vendor)) == string(vendor) {
				return nil
			}
			return fmt.Errorf("model %s/%s is disabled in catalog", vendor, modelID)
		}
		return fmt.Errorf("model %s/%s does not exist in catalog", vendor, modelID)
	}
	return fmt.Errorf("vendor %s does not exist in catalog", vendor)
}
