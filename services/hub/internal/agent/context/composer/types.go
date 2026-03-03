// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package composer

import "strings"

// ResourceType enumerates all mentionable resource domains in composer input.
type ResourceType string

const (
	ResourceTypeModel ResourceType = "model"
	ResourceTypeRule  ResourceType = "rule"
	ResourceTypeSkill ResourceType = "skill"
	ResourceTypeMCP   ResourceType = "mcp"
	ResourceTypeFile  ResourceType = "file"
)

// ParseResourceType parses and validates one resource type token.
func ParseResourceType(raw string) (ResourceType, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch ResourceType(normalized) {
	case ResourceTypeModel, ResourceTypeRule, ResourceTypeSkill, ResourceTypeMCP, ResourceTypeFile:
		return ResourceType(normalized), true
	default:
		return "", false
	}
}

// ResourceRef is one normalized resource mention reference.
type ResourceRef struct {
	Type ResourceType
	ID   string
}

// ParseResult is the normalized result of parsing one composer input string.
type ParseResult struct {
	RawInput      string
	TrimmedInput  string
	IsCommand     bool
	CommandText   string
	PromptText    string
	MentionedRefs []ResourceRef
}

// CommandKind describes the semantic type of one slash command.
type CommandKind string

const (
	CommandKindControl CommandKind = "control"
	CommandKindPrompt  CommandKind = "prompt"
)

// CommandMeta carries command metadata for suggestion rendering.
type CommandMeta struct {
	Name        string
	Description string
	Kind        CommandKind
}

// SuggestionKind distinguishes suggestions by semantic origin.
type SuggestionKind string

const (
	SuggestionKindCommand      SuggestionKind = "command"
	SuggestionKindResourceType SuggestionKind = "resource_type"
	SuggestionKindResource     SuggestionKind = "resource"
)

// ResourceCatalogItem is one resource entry available to composer suggestions.
type ResourceCatalogItem struct {
	Type ResourceType
	ID   string
	Name string
}

// Suggestion is one completion candidate.
type Suggestion struct {
	Kind         SuggestionKind
	Label        string
	Detail       string
	InsertText   string
	ReplaceStart int
	ReplaceEnd   int
}

// SuggestRequest describes one completion request.
type SuggestRequest struct {
	Draft     string
	Cursor    int
	Limit     int
	Commands  []CommandMeta
	Resources []ResourceCatalogItem
}
