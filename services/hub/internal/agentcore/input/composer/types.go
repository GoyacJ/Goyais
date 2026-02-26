package composer

import "strings"

type ResourceType string

const (
	ResourceTypeModel ResourceType = "model"
	ResourceTypeRule  ResourceType = "rule"
	ResourceTypeSkill ResourceType = "skill"
	ResourceTypeMCP   ResourceType = "mcp"
	ResourceTypeFile  ResourceType = "file"
)

func ParseResourceType(raw string) (ResourceType, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch ResourceType(normalized) {
	case ResourceTypeModel, ResourceTypeRule, ResourceTypeSkill, ResourceTypeMCP, ResourceTypeFile:
		return ResourceType(normalized), true
	default:
		return "", false
	}
}

type ResourceRef struct {
	Type ResourceType
	ID   string
}

type ParseResult struct {
	RawInput      string
	TrimmedInput  string
	IsCommand     bool
	CommandText   string
	PromptText    string
	MentionedRefs []ResourceRef
}

type CommandKind string

const (
	CommandKindControl CommandKind = "control"
	CommandKindPrompt  CommandKind = "prompt"
)

type CommandMeta struct {
	Name        string
	Description string
	Kind        CommandKind
}

type CommandDispatchResult struct {
	Name           string
	Kind           CommandKind
	Output         string
	ExpandedPrompt string
}

type SuggestionKind string

const (
	SuggestionKindCommand      SuggestionKind = "command"
	SuggestionKindResourceType SuggestionKind = "resource_type"
	SuggestionKindResource     SuggestionKind = "resource"
)

type ResourceCatalogItem struct {
	Type ResourceType
	ID   string
	Name string
}

type Suggestion struct {
	Kind         SuggestionKind
	Label        string
	Detail       string
	InsertText   string
	ReplaceStart int
	ReplaceEnd   int
}

type SuggestRequest struct {
	Draft     string
	Cursor    int
	Limit     int
	Commands  []CommandMeta
	Resources []ResourceCatalogItem
}
