package composer

import (
	"path"
	"regexp"
	"strings"
)

var resourceMentionPattern = regexp.MustCompile(`@(?P<type>model|rule|skill|mcp|file):(?P<id>[A-Za-z0-9_./-]+)`)

func Parse(raw string) ParseResult {
	trimmed := strings.TrimSpace(raw)
	result := ParseResult{
		RawInput:     raw,
		TrimmedInput: trimmed,
		PromptText:   StripResourceMentions(trimmed),
	}
	if trimmed == "" {
		return result
	}
	if strings.HasPrefix(trimmed, "/") {
		result.IsCommand = true
		result.CommandText = trimmed
		return result
	}
	result.MentionedRefs = ExtractResourceMentions(trimmed)
	return result
}

func ParsePrompt(raw string) ParseResult {
	trimmed := strings.TrimSpace(raw)
	return ParseResult{
		RawInput:      raw,
		TrimmedInput:  trimmed,
		PromptText:    StripResourceMentions(trimmed),
		MentionedRefs: ExtractResourceMentions(trimmed),
	}
}

func ExtractResourceMentions(raw string) []ResourceRef {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	matches := resourceMentionPattern.FindAllStringSubmatch(trimmed, -1)
	if len(matches) == 0 {
		return nil
	}

	typeIndex := resourceMentionPattern.SubexpIndex("type")
	idIndex := resourceMentionPattern.SubexpIndex("id")
	if typeIndex < 0 || idIndex < 0 {
		return nil
	}

	out := make([]ResourceRef, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		if len(match) <= idIndex || len(match) <= typeIndex {
			continue
		}
		typeValue, ok := ParseResourceType(match[typeIndex])
		if !ok {
			continue
		}
		id := strings.TrimSpace(match[idIndex])
		if id == "" {
			continue
		}
		key := strings.ToLower(string(typeValue) + ":" + id)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ResourceRef{Type: typeValue, ID: id})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func StripResourceMentions(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	typeIndex := resourceMentionPattern.SubexpIndex("type")
	idIndex := resourceMentionPattern.SubexpIndex("id")
	cleaned := resourceMentionPattern.ReplaceAllStringFunc(trimmed, func(match string) string {
		submatches := resourceMentionPattern.FindStringSubmatch(match)
		if len(submatches) == 0 || len(submatches) <= typeIndex || len(submatches) <= idIndex {
			return ""
		}
		resourceType, ok := ParseResourceType(submatches[typeIndex])
		if !ok {
			return ""
		}
		resourceID := strings.TrimSpace(submatches[idIndex])
		if resourceID == "" {
			return ""
		}
		if resourceType == ResourceTypeFile {
			normalized := path.Clean(strings.ReplaceAll(resourceID, "\\", "/"))
			normalized = strings.TrimPrefix(normalized, "./")
			if normalized == "." {
				return ""
			}
			return normalized
		}
		return ""
	})
	return strings.Join(strings.Fields(cleaned), " ")
}
