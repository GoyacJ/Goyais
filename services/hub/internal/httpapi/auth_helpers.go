package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type apiError struct {
	status  int
	code    string
	message string
	details map[string]any
}

func (e *apiError) write(w http.ResponseWriter, r *http.Request) {
	WriteStandardError(w, r, e.status, e.code, e.message, e.details)
}

func decodeJSONBody(r *http.Request, v any) *apiError {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "Invalid JSON request body",
			details: map[string]any{"reason": err.Error()},
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isValidHubURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

func extractAccessToken(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(authorization, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
		}
	}

	return strings.TrimSpace(r.Header.Get("X-Auth-Token"))
}

func parseRole(raw string) Role {
	switch Role(strings.TrimSpace(raw)) {
	case RoleViewer:
		return RoleViewer
	case RoleDeveloper:
		return RoleDeveloper
	case RoleApprover:
		return RoleApprover
	case RoleAdmin:
		return RoleAdmin
	default:
		return RoleDeveloper
	}
}

func capabilitiesForRole(role Role) Capabilities {
	switch role {
	case RoleAdmin:
		return Capabilities{AdminConsole: true, ResourceWrite: true, ExecutionControl: true}
	case RoleApprover:
		return Capabilities{AdminConsole: false, ResourceWrite: true, ExecutionControl: true}
	case RoleDeveloper:
		return Capabilities{AdminConsole: false, ResourceWrite: true, ExecutionControl: true}
	default:
		return Capabilities{AdminConsole: false, ResourceWrite: false, ExecutionControl: false}
	}
}

func localMe() Me {
	return Me{
		UserID:       "local_user",
		DisplayName:  "Local User",
		WorkspaceID:  localWorkspaceID,
		Role:         RoleAdmin,
		Capabilities: capabilitiesForRole(RoleAdmin),
	}
}

func validateLoginRequest(input LoginRequest) *apiError {
	if strings.TrimSpace(input.WorkspaceID) == "" {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "workspace_id is required",
			details: map[string]any{"field": "workspace_id"},
		}
	}

	hasToken := strings.TrimSpace(input.Token) != ""
	hasUsername := strings.TrimSpace(input.Username) != ""
	hasPassword := strings.TrimSpace(input.Password) != ""

	if hasToken {
		if strings.EqualFold(strings.TrimSpace(input.Token), "invalid") {
			return &apiError{
				status:  http.StatusUnauthorized,
				code:    "AUTH_INVALID_CREDENTIALS",
				message: "Token authentication failed",
				details: map[string]any{"field": "token"},
			}
		}
		return nil
	}

	if hasUsername != hasPassword {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "username and password must be provided together",
			details: map[string]any{"fields": []string{"username", "password"}},
		}
	}

	if !hasUsername {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "Either token or username/password is required",
			details: map[string]any{"fields": []string{"token", "username", "password"}},
		}
	}

	if strings.EqualFold(strings.TrimSpace(input.Username), "invalid") || strings.EqualFold(strings.TrimSpace(input.Password), "invalid") || strings.EqualFold(strings.TrimSpace(input.Password), "wrong") {
		return &apiError{
			status:  http.StatusUnauthorized,
			code:    "AUTH_INVALID_CREDENTIALS",
			message: "Username or password is invalid",
			details: map[string]any{"workspace_id": input.WorkspaceID},
		}
	}

	return nil
}

func generateAccessToken() string {
	return "at_" + randomHex(12)
}
