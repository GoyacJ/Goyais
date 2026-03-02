package policy

import (
	"sort"
	"strings"
)

type HookScope string

const (
	HookScopeGlobal  HookScope = "global"
	HookScopeProject HookScope = "project"
	HookScopeLocal   HookScope = "local"
	HookScopePlugin  HookScope = "plugin"
)

type HookPolicy struct {
	ID                string
	Scope             HookScope
	EventType         string
	ToolName          string
	WorkspaceID       string
	ProjectID         string
	ConversationID    string
	Action            string
	Reason            string
	Enabled           bool
	UpdatedInput      map[string]any
	AdditionalContext map[string]any
}

type HookScopeContext struct {
	WorkspaceID      string
	ProjectID        string
	ConversationID   string
	ToolName         string
	IsLocalWorkspace bool
}

func ResolveHookPolicies(policies []HookPolicy, context HookScopeContext) []HookPolicy {
	resolved := make([]HookPolicy, 0, len(policies))
	for _, item := range policies {
		if !item.Enabled {
			continue
		}
		if !policyAppliesToContext(item, context) {
			continue
		}
		copyItem := item
		copyItem.UpdatedInput = cloneMapAny(item.UpdatedInput)
		copyItem.AdditionalContext = cloneMapAny(item.AdditionalContext)
		resolved = append(resolved, copyItem)
	}
	sort.SliceStable(resolved, func(i int, j int) bool {
		left := hookScopeOrder(normalizeScope(resolved[i].Scope))
		right := hookScopeOrder(normalizeScope(resolved[j].Scope))
		if left != right {
			return left < right
		}
		return strings.TrimSpace(resolved[i].ID) < strings.TrimSpace(resolved[j].ID)
	})
	return resolved
}

func policyAppliesToContext(policy HookPolicy, context HookScopeContext) bool {
	normalizedScope := normalizeScope(policy.Scope)
	if !hasValidScopeBindings(policy, normalizedScope) {
		return false
	}
	if !matchesWorkspaceBinding(policy, context.WorkspaceID) {
		return false
	}
	switch normalizedScope {
	case HookScopeGlobal:
		return true
	case HookScopeProject:
		projectID := strings.TrimSpace(context.ProjectID)
		if projectID == "" {
			return false
		}
		return matchesProjectBinding(policy, projectID)
	case HookScopeLocal:
		if !context.IsLocalWorkspace {
			return false
		}
		conversationID := strings.TrimSpace(context.ConversationID)
		if conversationID == "" {
			return false
		}
		return matchesConversationBinding(policy, conversationID)
	case HookScopePlugin:
		return isPluginToolName(context.ToolName)
	default:
		return false
	}
}

func hasValidScopeBindings(policy HookPolicy, scope HookScope) bool {
	projectBinding := firstNonEmpty(strings.TrimSpace(policy.ProjectID), bindingValue(policy.AdditionalContext, "project_id"))
	conversationBinding := firstNonEmpty(strings.TrimSpace(policy.ConversationID), bindingValue(policy.AdditionalContext, "conversation_id"))
	switch scope {
	case HookScopeGlobal, HookScopePlugin:
		return projectBinding == "" && conversationBinding == ""
	case HookScopeProject:
		return projectBinding != "" && conversationBinding == ""
	case HookScopeLocal:
		return projectBinding == "" && conversationBinding != ""
	default:
		return false
	}
}

func matchesWorkspaceBinding(policy HookPolicy, workspaceID string) bool {
	expected := firstNonEmpty(strings.TrimSpace(policy.WorkspaceID), bindingValue(policy.AdditionalContext, "workspace_id"))
	if expected == "" {
		return true
	}
	return strings.EqualFold(expected, strings.TrimSpace(workspaceID))
}

func matchesProjectBinding(policy HookPolicy, projectID string) bool {
	expected := firstNonEmpty(strings.TrimSpace(policy.ProjectID), bindingValue(policy.AdditionalContext, "project_id"))
	if expected == "" {
		return true
	}
	return strings.EqualFold(expected, strings.TrimSpace(projectID))
}

func matchesConversationBinding(policy HookPolicy, conversationID string) bool {
	expected := firstNonEmpty(strings.TrimSpace(policy.ConversationID), bindingValue(policy.AdditionalContext, "conversation_id"))
	if expected == "" {
		return true
	}
	return strings.EqualFold(expected, strings.TrimSpace(conversationID))
}

func bindingValue(input map[string]any, key string) string {
	if len(input) == 0 {
		return ""
	}
	value, ok := input[strings.TrimSpace(key)]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func normalizeScope(value HookScope) HookScope {
	switch HookScope(strings.TrimSpace(string(value))) {
	case HookScopeGlobal:
		return HookScopeGlobal
	case HookScopeProject:
		return HookScopeProject
	case HookScopeLocal:
		return HookScopeLocal
	case HookScopePlugin:
		return HookScopePlugin
	default:
		return ""
	}
}

func hookScopeOrder(value HookScope) int {
	switch value {
	case HookScopeGlobal:
		return 0
	case HookScopeProject:
		return 1
	case HookScopeLocal:
		return 2
	case HookScopePlugin:
		return 3
	default:
		return 4
	}
}

func isPluginToolName(toolName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(normalized, "plugin.") || strings.HasPrefix(normalized, "plugin/") || strings.HasPrefix(normalized, "plugin_")
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
