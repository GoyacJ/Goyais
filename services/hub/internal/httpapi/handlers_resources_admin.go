package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

func ResourcesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"resource.read",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
			workspaceID = session.WorkspaceID
		}

		state.mu.RLock()
		items := make([]Resource, 0)
		for _, item := range state.resources {
			if workspaceID != "" && item.WorkspaceID != workspaceID {
				continue
			}
			items = append(items, item)
		}
		state.mu.RUnlock()
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })

		raw := make([]any, 0, len(items))
		for _, item := range items {
			raw = append(raw, item)
		}
		start, limit := parseCursorLimit(r)
		paged, next := paginateAny(raw, start, limit)
		writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
	}
}

func ResourceImportsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		input := ResourceImportRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(input.SourceID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id and source_id are required", map[string]any{})
			return
		}
		if input.ResourceType == "" {
			input.ResourceType = ResourceTypeModel
		}
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"resource.write",
			authorizationResource{
				WorkspaceID:  workspaceID,
				ResourceType: string(input.ResourceType),
				Scope:        "private",
			},
			authorizationContext{OperationType: "write"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		resource := Resource{
			ID:          "res_" + randomHex(6),
			WorkspaceID: workspaceID,
			Type:        input.ResourceType,
			Name:        "Imported " + strings.ToUpper(string(input.ResourceType)),
			Source:      "local_import",
			Scope:       "private",
			ShareStatus: ShareStatusPending,
			OwnerUserID: "local_user",
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		state.mu.Lock()
		state.resources[resource.ID] = resource
		state.mu.Unlock()

		writeJSON(w, http.StatusCreated, resource)
		state.AppendAudit(AdminAuditEvent{Actor: actorFromSession(session), Action: "resource.import", Resource: resource.ID, Result: "success", TraceID: TraceIDFromContext(r.Context())})
		if state.authz != nil {
			_ = state.authz.appendAudit(workspaceID, session.UserID, "resource.import", "resource", resource.ID, "success", map[string]any{
				"resource_type": input.ResourceType,
				"source_id":     input.SourceID,
			}, TraceIDFromContext(r.Context()))
		}
	}
}

func ShareRequestsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		payload := struct {
			ResourceID string `json:"resource_id"`
		}{}
		if err := decodeJSONBody(r, &payload); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(payload.ResourceID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "resource_id is required", map[string]any{})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"share.request",
			authorizationResource{
				WorkspaceID: workspaceID,
				OwnerUserID: "",
			},
			authorizationContext{OperationType: "write"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		request := ShareRequest{
			ID:              "share_" + randomHex(6),
			WorkspaceID:     workspaceID,
			ResourceID:      strings.TrimSpace(payload.ResourceID),
			Status:          ShareStatusPending,
			RequesterUserID: session.UserID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		state.mu.Lock()
		state.shareRequests[request.ID] = request
		state.mu.Unlock()
		writeJSON(w, http.StatusCreated, request)
		state.AppendAudit(AdminAuditEvent{Actor: actorFromSession(session), Action: "share_request.create", Resource: request.ID, Result: "success", TraceID: TraceIDFromContext(r.Context())})
		if state.authz != nil {
			_ = state.authz.appendAudit(workspaceID, session.UserID, "share.request", "share_request", request.ID, "success", map[string]any{
				"resource_id": payload.ResourceID,
			}, TraceIDFromContext(r.Context()))
		}
	}
}

func ShareRequestActionHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		requestID := strings.TrimSpace(r.PathValue("request_id"))
		action := strings.TrimSpace(r.PathValue("action"))

		state.mu.RLock()
		request, exists := state.shareRequests[requestID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "SHARE_REQUEST_NOT_FOUND", "Share request does not exist", map[string]any{"request_id": requestID})
			return
		}

		actionKey := "share." + action
		allowedRoles := []Role{RoleAdmin, RoleApprover}
		if action == "revoke" {
			allowedRoles = []Role{RoleAdmin, RoleApprover, RoleDeveloper}
		}
		session, authErr := authorizeAction(
			state,
			r,
			request.WorkspaceID,
			actionKey,
			authorizationResource{
				WorkspaceID: request.WorkspaceID,
				ShareStatus: string(request.Status),
			},
			authorizationContext{OperationType: "write", ABACRequired: true},
			allowedRoles...,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		request = state.shareRequests[requestID]
		switch action {
		case "approve":
			request.Status = ShareStatusApproved
		case "reject":
			request.Status = ShareStatusDenied
		case "revoke":
			request.Status = ShareStatusRevoked
		default:
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route does not exist", map[string]any{"action": action})
			return
		}
		approver := session.UserID
		request.ApproverUserID = &approver
		request.UpdatedAt = now
		state.shareRequests[requestID] = request
		state.mu.Unlock()

		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "share_request." + action,
			Resource: requestID,
			Result:   "success",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(request.WorkspaceID, session.UserID, actionKey, "share_request", requestID, "success", map[string]any{
				"new_status": request.Status,
			}, TraceIDFromContext(r.Context()))
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func ModelCatalogHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		traceID := TraceIDFromContext(r.Context())
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
			response, _, err := state.loadModelCatalogDetailed(workspaceID, false)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "MODEL_CATALOG_LOAD_FAILED", "Failed to load model catalog", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusOK, response)
		case http.MethodPost:
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.write",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin, RoleApprover, RoleDeveloper,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			input := struct {
				Source string `json:"source,omitempty"`
			}{}
			if strings.TrimSpace(r.Header.Get("Content-Type")) != "" && r.ContentLength != 0 {
				if err := decodeJSONBody(r, &input); err != nil {
					err.write(w, r)
					return
				}
			}
			trigger := normalizeCatalogReloadTrigger(input.Source)

			response, meta, err := state.loadModelCatalogDetailed(workspaceID, true)
			state.recordModelCatalogReloadAudit(workspaceID, trigger, meta, err, session.UserID, traceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "MODEL_CATALOG_RELOAD_FAILED", "Failed to reload model catalog", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusOK, response)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func AdminUsersHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"admin.users.manage",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read", ABACRequired: true},
				RoleAdmin,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
				workspaceID = session.WorkspaceID
			}

			if state.authz != nil {
				items, err := state.authz.listUsers(workspaceID)
				if err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users", map[string]any{})
					return
				}
				raw := make([]any, 0, len(items))
				for _, item := range items {
					raw = append(raw, item)
				}
				start, limit := parseCursorLimit(r)
				paged, next := paginateAny(raw, start, limit)
				writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
				return
			}

			state.mu.RLock()
			items := make([]AdminUser, 0)
			for _, user := range state.adminUsers {
				if workspaceID != "" && user.WorkspaceID != workspaceID {
					continue
				}
				items = append(items, user)
			}
			state.mu.RUnlock()
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			input := AdminUser{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			workspaceID := strings.TrimSpace(input.WorkspaceID)
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"admin.users.manage",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
				workspaceID = session.WorkspaceID
			}
			if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(input.Username) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id and username are required", map[string]any{})
				return
			}
			input.WorkspaceID = workspaceID
			if state.authz != nil {
				updated, err := state.authz.upsertUser(input)
				if err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert user", map[string]any{})
					return
				}
				writeJSON(w, http.StatusOK, updated)
				_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.users.manage", "user", updated.ID, "success", map[string]any{
					"username": updated.Username,
				}, TraceIDFromContext(r.Context()))
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			state.mu.Lock()
			for id, user := range state.adminUsers {
				if user.WorkspaceID == input.WorkspaceID && user.Username == input.Username {
					user.DisplayName = input.DisplayName
					if input.Role != "" {
						user.Role = input.Role
					}
					state.adminUsers[id] = user
					state.mu.Unlock()
					writeJSON(w, http.StatusOK, user)
					return
				}
			}
			if input.Role == "" {
				input.Role = RoleDeveloper
			}
			created := AdminUser{
				ID:          "u_" + randomHex(6),
				WorkspaceID: input.WorkspaceID,
				Username:    input.Username,
				DisplayName: firstNonEmpty(strings.TrimSpace(input.DisplayName), input.Username),
				Role:        input.Role,
				Enabled:     true,
				CreatedAt:   now,
			}
			state.adminUsers[created.ID] = created
			state.mu.Unlock()
			writeJSON(w, http.StatusCreated, created)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func AdminUserByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.PathValue("user_id"))
		workspaceID := ""
		if state.authz != nil {
			user, err := state.authz.getUserByID(userID)
			if err == nil {
				workspaceID = user.WorkspaceID
			}
		}
		if workspaceID == "" {
			state.mu.RLock()
			if user, exists := state.adminUsers[userID]; exists {
				workspaceID = user.WorkspaceID
			}
			state.mu.RUnlock()
		}
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"admin.users.manage",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
			RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		switch r.Method {
		case http.MethodDelete:
			if state.authz != nil {
				if err := state.authz.deleteUser(userID); err != nil {
					WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
					return
				}
				writeJSON(w, http.StatusNoContent, map[string]any{})
				_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.users.manage", "user", userID, "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
				return
			}
			state.mu.Lock()
			if _, exists := state.adminUsers[userID]; !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
				return
			}
			delete(state.adminUsers, userID)
			state.mu.Unlock()
			writeJSON(w, http.StatusNoContent, map[string]any{})
		case http.MethodPatch:
			payload := struct {
				Enabled *bool `json:"enabled,omitempty"`
				Role    Role  `json:"role,omitempty"`
			}{}
			if err := decodeJSONBody(r, &payload); err != nil {
				err.write(w, r)
				return
			}
			if state.authz != nil {
				if payload.Enabled != nil {
					updated, err := state.authz.setUserEnabled(userID, *payload.Enabled)
					if err != nil {
						WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
						return
					}
					if payload.Role != "" {
						updated, err = state.authz.setUserRole(userID, payload.Role)
						if err != nil {
							WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
							return
						}
					}
					writeJSON(w, http.StatusOK, updated)
					_ = state.authz.appendAudit(updated.WorkspaceID, session.UserID, "admin.users.manage", "user", userID, "success", map[string]any{"operation": "patch"}, TraceIDFromContext(r.Context()))
					return
				}
				if payload.Role != "" {
					updated, err := state.authz.setUserRole(userID, payload.Role)
					if err != nil {
						WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
						return
					}
					writeJSON(w, http.StatusOK, updated)
					_ = state.authz.appendAudit(updated.WorkspaceID, session.UserID, "admin.users.manage", "user", userID, "success", map[string]any{"operation": "patch"}, TraceIDFromContext(r.Context()))
					return
				}
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "role or enabled is required", map[string]any{})
				return
			}
			state.mu.Lock()
			user, exists := state.adminUsers[userID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User does not exist", map[string]any{"user_id": userID})
				return
			}
			if payload.Enabled != nil {
				user.Enabled = *payload.Enabled
			}
			if payload.Role != "" {
				user.Role = payload.Role
			}
			state.adminUsers[userID] = user
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, user)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func AdminRolesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			session, authErr := authorizeAction(
				state,
				r,
				"",
				"admin.roles.manage",
				authorizationResource{},
				authorizationContext{OperationType: "read", ABACRequired: true},
				RoleAdmin,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			workspaceID := firstNonEmpty(strings.TrimSpace(r.URL.Query().Get("workspace_id")), session.WorkspaceID)
			if state.authz != nil {
				items, err := state.authz.listRoles(workspaceID)
				if err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list roles", map[string]any{})
					return
				}
				writeJSON(w, http.StatusOK, items)
				return
			}
			state.mu.RLock()
			items := make([]AdminRole, 0, len(state.adminRoles))
			for _, role := range state.adminRoles {
				items = append(items, role)
			}
			state.mu.RUnlock()
			sort.Slice(items, func(i, j int) bool { return strings.Compare(string(items[i].Key), string(items[j].Key)) < 0 })
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			input := AdminRole{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			session, authErr := authorizeAction(
				state,
				r,
				"",
				"admin.roles.manage",
				authorizationResource{},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if input.Key == "" || strings.TrimSpace(input.Name) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "key and name are required", map[string]any{})
				return
			}
			workspaceID := firstNonEmpty(strings.TrimSpace(r.URL.Query().Get("workspace_id")), session.WorkspaceID)
			if state.authz != nil {
				updated, err := state.authz.upsertRole(workspaceID, input)
				if err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert role", map[string]any{})
					return
				}
				writeJSON(w, http.StatusOK, updated)
				_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.roles.manage", "role", string(updated.Key), "success", map[string]any{"operation": "upsert"}, TraceIDFromContext(r.Context()))
				return
			}
			state.mu.Lock()
			state.adminRoles[input.Key] = input
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, input)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func AdminRoleByKeyHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleKey := Role(strings.TrimSpace(r.PathValue("role_key")))
		session, authErr := authorizeAction(
			state,
			r,
			"",
			"admin.roles.manage",
			authorizationResource{},
			authorizationContext{OperationType: "write", ABACRequired: true},
			RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		workspaceID := firstNonEmpty(strings.TrimSpace(r.URL.Query().Get("workspace_id")), session.WorkspaceID)
		switch r.Method {
		case http.MethodDelete:
			if state.authz != nil {
				if err := state.authz.deleteRole(workspaceID, roleKey); err != nil {
					WriteStandardError(w, r, http.StatusNotFound, "ROLE_NOT_FOUND", "Role does not exist", map[string]any{"role": roleKey})
					return
				}
				writeJSON(w, http.StatusNoContent, map[string]any{})
				_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.roles.manage", "role", string(roleKey), "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
				return
			}
			state.mu.Lock()
			if _, exists := state.adminRoles[roleKey]; !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "ROLE_NOT_FOUND", "Role does not exist", map[string]any{"role": roleKey})
				return
			}
			delete(state.adminRoles, roleKey)
			state.mu.Unlock()
			writeJSON(w, http.StatusNoContent, map[string]any{})
		case http.MethodPatch:
			payload := struct {
				Enabled bool `json:"enabled"`
			}{}
			if err := decodeJSONBody(r, &payload); err != nil {
				err.write(w, r)
				return
			}
			if state.authz != nil {
				updated, err := state.authz.setRoleEnabled(workspaceID, roleKey, payload.Enabled)
				if err != nil {
					WriteStandardError(w, r, http.StatusNotFound, "ROLE_NOT_FOUND", "Role does not exist", map[string]any{"role": roleKey})
					return
				}
				writeJSON(w, http.StatusOK, updated)
				_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.roles.manage", "role", string(roleKey), "success", map[string]any{"operation": "patch_enabled"}, TraceIDFromContext(r.Context()))
				return
			}
			state.mu.Lock()
			role, exists := state.adminRoles[roleKey]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "ROLE_NOT_FOUND", "Role does not exist", map[string]any{"role": roleKey})
				return
			}
			role.Enabled = payload.Enabled
			state.adminRoles[roleKey] = role
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, role)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func AdminAuditHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"admin.audit.read",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "read"},
			RoleAdmin, RoleApprover,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
			workspaceID = session.WorkspaceID
		}
		if state.authz != nil {
			items, err := state.authz.listAudit(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list audit logs", map[string]any{})
				return
			}
			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
			return
		}
		state.mu.RLock()
		items := make([]any, 0)
		for _, audit := range state.adminAudit {
			if workspaceID != "" && !strings.Contains(audit.Resource, workspaceID) && audit.Resource != workspaceID {
				// 审计资源并不总是 workspace_id，这里做最小过滤。
			}
			items = append(items, audit)
		}
		state.mu.RUnlock()
		start, limit := parseCursorLimit(r)
		paged, next := paginateAny(items, start, limit)
		writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
	}
}
