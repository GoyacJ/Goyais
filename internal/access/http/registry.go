package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"goyais/internal/common/errorx"
	"goyais/internal/registry"
)

func (h *apiHandler) handleRegistryCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.registryService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
		return
	}

	query := r.URL.Query()
	cursor := strings.TrimSpace(query.Get("cursor"))
	page := 1
	pageSize := 20
	if cursor == "" {
		if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
			parsed, err := strconv.Atoi(rawPage)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"page": rawPage})
				return
			}
			page = parsed
		}
		if rawPageSize := strings.TrimSpace(query.Get("pageSize")); rawPageSize != "" {
			parsed, err := strconv.Atoi(rawPageSize)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"pageSize": rawPageSize})
				return
			}
			pageSize = parsed
		}
	}

	result, err := h.registryService.ListCapabilities(r.Context(), registry.ListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeRegistryError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toCapabilityPayload(item))
	}

	resp := map[string]any{"items": items}
	if result.UsedCursor {
		cursorInfo := map[string]any{"nextCursor": nil}
		if result.NextCursor != "" {
			cursorInfo["nextCursor"] = result.NextCursor
		}
		resp["cursorInfo"] = cursorInfo
	} else {
		resp["pageInfo"] = map[string]any{
			"page":     page,
			"pageSize": pageSize,
			"total":    result.Total,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *apiHandler) handleRegistryCapabilityRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	capabilityID := pathID("/api/v1/registry/capabilities/", r.URL.Path)
	if capabilityID == "" {
		errorx.Write(w, http.StatusNotFound, "REGISTRY_NOT_FOUND", "error.registry.not_found", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.registryService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
		return
	}

	item, err := h.registryService.GetCapability(r.Context(), reqCtx, capabilityID)
	if err != nil {
		writeRegistryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCapabilityPayload(item))
}

func (h *apiHandler) handleRegistryAlgorithms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.registryService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
		return
	}

	query := r.URL.Query()
	cursor := strings.TrimSpace(query.Get("cursor"))
	page := 1
	pageSize := 20
	if cursor == "" {
		if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
			parsed, err := strconv.Atoi(rawPage)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"page": rawPage})
				return
			}
			page = parsed
		}
		if rawPageSize := strings.TrimSpace(query.Get("pageSize")); rawPageSize != "" {
			parsed, err := strconv.Atoi(rawPageSize)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"pageSize": rawPageSize})
				return
			}
			pageSize = parsed
		}
	}

	result, err := h.registryService.ListAlgorithms(r.Context(), registry.ListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeRegistryError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAlgorithmPayload(item))
	}

	resp := map[string]any{"items": items}
	if result.UsedCursor {
		cursorInfo := map[string]any{"nextCursor": nil}
		if result.NextCursor != "" {
			cursorInfo["nextCursor"] = result.NextCursor
		}
		resp["cursorInfo"] = cursorInfo
	} else {
		resp["pageInfo"] = map[string]any{
			"page":     page,
			"pageSize": pageSize,
			"total":    result.Total,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *apiHandler) handleRegistryAlgorithmRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	algorithmID := pathID("/api/v1/registry/algorithms/", r.URL.Path)
	if algorithmID == "" {
		errorx.Write(w, http.StatusNotFound, "REGISTRY_NOT_FOUND", "error.registry.not_found", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.registryService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
		return
	}

	item, err := h.registryService.GetAlgorithm(r.Context(), reqCtx, algorithmID)
	if err != nil {
		writeRegistryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAlgorithmPayload(item))
}

func (h *apiHandler) handleRegistryProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.registryService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
		return
	}

	query := r.URL.Query()
	cursor := strings.TrimSpace(query.Get("cursor"))
	page := 1
	pageSize := 20
	if cursor == "" {
		if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
			parsed, err := strconv.Atoi(rawPage)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"page": rawPage})
				return
			}
			page = parsed
		}
		if rawPageSize := strings.TrimSpace(query.Get("pageSize")); rawPageSize != "" {
			parsed, err := strconv.Atoi(rawPageSize)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"pageSize": rawPageSize})
				return
			}
			pageSize = parsed
		}
	}

	result, err := h.registryService.ListProviders(r.Context(), registry.ListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeRegistryError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toProviderPayload(item))
	}

	resp := map[string]any{"items": items}
	if result.UsedCursor {
		cursorInfo := map[string]any{"nextCursor": nil}
		if result.NextCursor != "" {
			cursorInfo["nextCursor"] = result.NextCursor
		}
		resp["cursorInfo"] = cursorInfo
	} else {
		resp["pageInfo"] = map[string]any{
			"page":     page,
			"pageSize": pageSize,
			"total":    result.Total,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeRegistryError(w http.ResponseWriter, err error) {
	var forbidden *registry.ForbiddenError
	switch {
	case errors.Is(err, registry.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_REGISTRY_REQUEST", "error.registry.invalid_request", nil)
	case errors.Is(err, registry.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, registry.ErrCapabilityNotFound):
		errorx.Write(w, http.StatusNotFound, "CAPABILITY_NOT_FOUND", "error.registry.not_found", nil)
	case errors.Is(err, registry.ErrAlgorithmNotFound):
		errorx.Write(w, http.StatusNotFound, "ALGORITHM_NOT_FOUND", "error.registry.not_found", nil)
	case errors.As(err, &forbidden), errors.Is(err, registry.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	case errors.Is(err, registry.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.registry.not_implemented", nil)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func toCapabilityPayload(item registry.Capability) map[string]any {
	resp := map[string]any{
		"id":                  item.ID,
		"tenantId":            item.TenantID,
		"workspaceId":         item.WorkspaceID,
		"ownerId":             item.OwnerID,
		"visibility":          item.Visibility,
		"acl":                 decodeJSON(item.ACLJSON, []any{}),
		"status":              item.Status,
		"name":                item.Name,
		"kind":                item.Kind,
		"version":             item.Version,
		"providerId":          item.ProviderID,
		"inputSchema":         decodeJSON(item.InputSchemaJSON, map[string]any{}),
		"outputSchema":        decodeJSON(item.OutputSchemaJSON, map[string]any{}),
		"requiredPermissions": decodeJSON(item.RequiredPermissionsJSON, []any{}),
		"egressPolicy":        decodeJSON(item.EgressPolicyJSON, map[string]any{}),
		"createdAt":           item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":           item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	return resp
}

func toAlgorithmPayload(item registry.Algorithm) map[string]any {
	resp := map[string]any{
		"id":           item.ID,
		"tenantId":     item.TenantID,
		"workspaceId":  item.WorkspaceID,
		"ownerId":      item.OwnerID,
		"visibility":   item.Visibility,
		"acl":          decodeJSON(item.ACLJSON, []any{}),
		"status":       item.Status,
		"name":         item.Name,
		"version":      item.Version,
		"templateRef":  item.TemplateRef,
		"defaults":     decodeJSON(item.DefaultsJSON, map[string]any{}),
		"constraints":  decodeJSON(item.ConstraintsJSON, map[string]any{}),
		"dependencies": decodeJSON(item.DependenciesJSON, map[string]any{}),
		"createdAt":    item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":    item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	return resp
}

func toProviderPayload(item registry.CapabilityProvider) map[string]any {
	resp := map[string]any{
		"id":           item.ID,
		"tenantId":     item.TenantID,
		"workspaceId":  item.WorkspaceID,
		"ownerId":      item.OwnerID,
		"visibility":   item.Visibility,
		"acl":          decodeJSON(item.ACLJSON, []any{}),
		"status":       item.Status,
		"name":         item.Name,
		"providerType": item.ProviderType,
		"endpoint":     item.Endpoint,
		"metadata":     decodeJSON(item.MetadataJSON, map[string]any{}),
		"createdAt":    item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":    item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	return resp
}
