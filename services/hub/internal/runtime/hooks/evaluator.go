package hooks

import (
	"sort"
	"strings"
)

type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
	ScopeLocal   Scope = "local"
	ScopePlugin  Scope = "plugin"
)

type EventType string

const (
	EventTypeSessionStart       EventType = "session_start"
	EventTypeUserPromptSubmit   EventType = "user_prompt_submit"
	EventTypePreToolUse         EventType = "pre_tool_use"
	EventTypePermissionRequest  EventType = "permission_request"
	EventTypePostToolUse        EventType = "post_tool_use"
	EventTypePostToolUseFailure EventType = "post_tool_use_failure"
	EventTypeStop               EventType = "stop"
	EventTypeSubagentStop       EventType = "subagent_stop"
	EventTypeNotification       EventType = "notification"
	EventTypeConfigChange       EventType = "config_change"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
)

type Policy struct {
	ID                string
	Scope             Scope
	EventType         EventType
	ToolName          string
	Action            Action
	Reason            string
	Enabled           bool
	UpdatedInput      map[string]any
	AdditionalContext map[string]any
}

type EventInput struct {
	EventType EventType
	ToolName  string
}

type Decision struct {
	Action            Action
	PolicyID          string
	Scope             Scope
	Reason            string
	UpdatedInput      map[string]any
	AdditionalContext map[string]any
}

func Evaluate(policies []Policy, event EventInput) Decision {
	filtered := make([]Policy, 0, len(policies))
	eventTool := strings.TrimSpace(event.ToolName)
	for _, item := range policies {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(string(item.EventType)) != strings.TrimSpace(string(event.EventType)) {
			continue
		}
		policyTool := strings.TrimSpace(item.ToolName)
		if policyTool != "" && !strings.EqualFold(policyTool, eventTool) {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return Decision{
			Action:            ActionAllow,
			UpdatedInput:      map[string]any{},
			AdditionalContext: map[string]any{},
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		left := normalizeScope(filtered[i].Scope)
		right := normalizeScope(filtered[j].Scope)
		if left != right {
			return scopeOrder(left) < scopeOrder(right)
		}
		leftSpecificity := toolSpecificity(filtered[i].ToolName, eventTool)
		rightSpecificity := toolSpecificity(filtered[j].ToolName, eventTool)
		if leftSpecificity != rightSpecificity {
			return leftSpecificity < rightSpecificity
		}
		leftAction := normalizeAction(filtered[i].Action)
		rightAction := normalizeAction(filtered[j].Action)
		if leftAction != rightAction {
			return actionOrder(leftAction) < actionOrder(rightAction)
		}
		return strings.TrimSpace(filtered[i].ID) < strings.TrimSpace(filtered[j].ID)
	})

	selected := filtered[0]
	action := normalizeAction(selected.Action)
	if action == "" {
		action = ActionAllow
	}
	return Decision{
		Action:            action,
		PolicyID:          strings.TrimSpace(selected.ID),
		Scope:             normalizeScope(selected.Scope),
		Reason:            strings.TrimSpace(selected.Reason),
		UpdatedInput:      cloneMapAny(selected.UpdatedInput),
		AdditionalContext: cloneMapAny(selected.AdditionalContext),
	}
}

func normalizeScope(value Scope) Scope {
	normalized := Scope(strings.TrimSpace(string(value)))
	switch normalized {
	case ScopeGlobal:
		return ScopeGlobal
	case ScopeProject:
		return ScopeProject
	case ScopeLocal:
		return ScopeLocal
	case ScopePlugin:
		return ScopePlugin
	default:
		return normalized
	}
}

func normalizeAction(value Action) Action {
	switch Action(strings.TrimSpace(string(value))) {
	case ActionAllow:
		return ActionAllow
	case ActionDeny:
		return ActionDeny
	case ActionAsk:
		return ActionAsk
	default:
		return ""
	}
}

func scopeOrder(scope Scope) int {
	switch scope {
	case ScopeGlobal:
		return 0
	case ScopeProject:
		return 1
	case ScopeLocal:
		return 2
	case ScopePlugin:
		return 3
	default:
		return 4
	}
}

func toolSpecificity(policyToolName string, eventToolName string) int {
	policyTool := strings.TrimSpace(policyToolName)
	eventTool := strings.TrimSpace(eventToolName)
	if policyTool == "" {
		return 1
	}
	if strings.EqualFold(policyTool, eventTool) {
		return 0
	}
	return 2
}

func actionOrder(action Action) int {
	switch action {
	case ActionDeny:
		return 0
	case ActionAsk:
		return 1
	case ActionAllow:
		return 2
	default:
		return 3
	}
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
