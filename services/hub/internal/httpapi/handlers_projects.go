package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
)

func ProjectsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"project.read",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if workspaceID == "" {
				workspaceID = session.WorkspaceID
			}
			items, err := listProjectsFromStore(state, workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_LIST_FAILED", "Failed to list projects", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			sort.Slice(items, func(i, j int) bool {
				if items[i].CreatedAt == items[j].CreatedAt {
					if items[i].UpdatedAt == items[j].UpdatedAt {
						return items[i].ID > items[j].ID
					}
					return items[i].UpdatedAt > items[j].UpdatedAt
				}
				return items[i].CreatedAt > items[j].CreatedAt
			})

			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			input := CreateProjectRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			if strings.TrimSpace(input.WorkspaceID) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.RepoPath) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id/name/repo_path are required", map[string]any{})
				return
			}
			session, authErr := authorizeAction(
				state,
				r,
				strings.TrimSpace(input.WorkspaceID),
				"project.write",
				authorizationResource{WorkspaceID: strings.TrimSpace(input.WorkspaceID)},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			project := Project{
				ID:                   "proj_" + randomHex(6),
				WorkspaceID:          strings.TrimSpace(input.WorkspaceID),
				Name:                 strings.TrimSpace(input.Name),
				RepoPath:             strings.TrimSpace(input.RepoPath),
				IsGit:                input.IsGit,
				DefaultModelConfigID: "",
				DefaultMode:          ConversationModeAgent,
				CurrentRevision:      0,
				CreatedAt:            now,
				UpdatedAt:            now,
			}
			savedProject, err := saveProjectToStore(state, project)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CREATE_FAILED", "Failed to create project", map[string]any{})
				return
			}
			defaultConfig := defaultProjectConfig(savedProject.ID, savedProject.DefaultModelConfigID, now)
			if _, err := saveProjectConfigToStore(state, savedProject.WorkspaceID, defaultConfig); err != nil {
				_, _ = deleteProjectFromStore(state, savedProject.ID)
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_CREATE_FAILED", "Failed to initialize project config", map[string]any{})
				return
			}

			writeJSON(w, http.StatusCreated, savedProject)
			state.AppendAudit(AdminAuditEvent{
				Actor:    actorFromSession(session),
				Action:   "project.create",
				Resource: savedProject.ID,
				Result:   "success",
				TraceID:  TraceIDFromContext(r.Context()),
			})
			if state.authz != nil {
				_ = state.authz.appendAudit(savedProject.WorkspaceID, session.UserID, "project.write", "project", savedProject.ID, "success", map[string]any{
					"operation": "create",
				}, TraceIDFromContext(r.Context()))
			}
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func ProjectsImportHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		input := ImportProjectRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.WorkspaceID) == "" || strings.TrimSpace(input.DirectoryPath) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id and directory_path are required", map[string]any{})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			strings.TrimSpace(input.WorkspaceID),
			"project.write",
			authorizationResource{WorkspaceID: strings.TrimSpace(input.WorkspaceID)},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		project := Project{
			ID:                   "proj_" + randomHex(6),
			WorkspaceID:          strings.TrimSpace(input.WorkspaceID),
			Name:                 deriveProjectName(input.DirectoryPath),
			RepoPath:             strings.TrimSpace(input.DirectoryPath),
			IsGit:                true,
			DefaultModelConfigID: "",
			DefaultMode:          ConversationModeAgent,
			CurrentRevision:      0,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		savedProject, err := saveProjectToStore(state, project)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_IMPORT_FAILED", "Failed to import project", map[string]any{})
			return
		}
		defaultConfig := defaultProjectConfig(savedProject.ID, savedProject.DefaultModelConfigID, now)
		if _, err := saveProjectConfigToStore(state, savedProject.WorkspaceID, defaultConfig); err != nil {
			_, _ = deleteProjectFromStore(state, savedProject.ID)
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_CREATE_FAILED", "Failed to initialize project config", map[string]any{})
			return
		}

		writeJSON(w, http.StatusCreated, savedProject)
		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "project.import_directory",
			Resource: savedProject.ID,
			Result:   "success",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(savedProject.WorkspaceID, session.UserID, "project.write", "project", savedProject.ID, "success", map[string]any{
				"operation": "import_directory",
			}, TraceIDFromContext(r.Context()))
		}
	}
}

func ProjectByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		projectID := strings.TrimSpace(r.PathValue("project_id"))
		project, exists, err := getProjectFromStore(state, projectID)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": projectID,
			})
			return
		}
		workspaceID := ""
		if exists {
			workspaceID = project.WorkspaceID
		}
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"project.write",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		deletedProject, err := deleteProjectFromStore(state, projectID)
		if errors.Is(err, sql.ErrNoRows) {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
				"project_id": projectID,
			})
			return
		}
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_DELETE_FAILED", "Failed to delete project", map[string]any{
				"project_id": projectID,
			})
			return
		}
		syncExecutionDomainBestEffort(state)

		writeJSON(w, http.StatusNoContent, map[string]any{})
		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "project.delete",
			Resource: deletedProject.ID,
			Result:   "success",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(deletedProject.WorkspaceID, session.UserID, "project.write", "project", deletedProject.ID, "success", map[string]any{
				"operation": "delete",
			}, TraceIDFromContext(r.Context()))
		}
	}
}

func deriveProjectName(path string) string {
	parts := strings.Split(strings.TrimSpace(path), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return parts[i]
		}
	}
	return "Imported Project"
}

func toStringPtr(value string) *string {
	copy := value
	return &copy
}
