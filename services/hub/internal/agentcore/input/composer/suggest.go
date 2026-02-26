package composer

import (
	"path"
	"sort"
	"strings"
)

func Suggest(req SuggestRequest) []Suggestion {
	draft := req.Draft
	cursor := req.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(draft) {
		cursor = len(draft)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 12
	}

	start, end := tokenBounds(draft, cursor)
	if start >= len(draft) && end >= len(draft) {
		return nil
	}
	tokenPrefix := strings.TrimSpace(draft[start:cursor])
	if tokenPrefix == "" {
		return nil
	}

	out := make([]Suggestion, 0, limit)
	if strings.HasPrefix(tokenPrefix, "/") {
		query := strings.ToLower(strings.TrimPrefix(tokenPrefix, "/"))
		for _, command := range req.Commands {
			score, ok := completionScore(command.Name, query)
			if !ok {
				continue
			}
			out = append(out, Suggestion{
				Kind:         SuggestionKindCommand,
				Label:        "/" + command.Name,
				Detail:       strings.TrimSpace(command.Description),
				InsertText:   "/" + command.Name,
				ReplaceStart: start,
				ReplaceEnd:   end,
			})
			_ = score
		}
		sortSuggestions(out)
		if len(out) > limit {
			return out[:limit]
		}
		return out
	}

	if !strings.HasPrefix(tokenPrefix, "@") {
		return nil
	}

	query := strings.TrimPrefix(tokenPrefix, "@")
	typePart, idPart, hasType := strings.Cut(query, ":")
	if !hasType {
		for _, candidate := range []string{"model", "rule", "skill", "mcp", "file"} {
			if _, ok := completionScore(candidate, strings.ToLower(strings.TrimSpace(typePart))); !ok {
				continue
			}
			out = append(out, Suggestion{
				Kind:         SuggestionKindResourceType,
				Label:        "@" + candidate + ":",
				Detail:       "",
				InsertText:   "@" + candidate + ":",
				ReplaceStart: start,
				ReplaceEnd:   end,
			})
		}
		sortSuggestions(out)
		if len(out) > limit {
			return out[:limit]
		}
		return out
	}

	resourceType, ok := ParseResourceType(typePart)
	if !ok {
		return nil
	}
	queryID := strings.ToLower(strings.TrimSpace(idPart))
	for _, resource := range req.Resources {
		if resource.Type != resourceType {
			continue
		}
		id := strings.TrimSpace(resource.ID)
		if id == "" {
			continue
		}
		if _, ok := completionScore(id, queryID); !ok {
			if resourceType == ResourceTypeFile {
				baseName := strings.ToLower(strings.TrimSpace(path.Base(id)))
				if _, ok := completionScore(baseName, queryID); ok {
					goto matchedResource
				}
			}
			if _, ok := completionScore(strings.ToLower(resource.Name), queryID); !ok {
				continue
			}
		}
	matchedResource:
		detail := ""
		if resourceType != ResourceTypeFile {
			normalizedName := strings.TrimSpace(resource.Name)
			if normalizedName != "" && !strings.EqualFold(normalizedName, id) {
				detail = normalizedName
			}
		}
		out = append(out, Suggestion{
			Kind:         SuggestionKindResource,
			Label:        "@" + string(resource.Type) + ":" + id,
			Detail:       detail,
			InsertText:   "@" + string(resource.Type) + ":" + id,
			ReplaceStart: start,
			ReplaceEnd:   end,
		})
	}

	sortSuggestions(out)
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func tokenBounds(draft string, cursor int) (int, int) {
	start := cursor
	for start > 0 {
		if isTokenBoundary(draft[start-1]) {
			break
		}
		start--
	}
	end := cursor
	for end < len(draft) {
		if isTokenBoundary(draft[end]) {
			break
		}
		end++
	}
	return start, end
}

func isTokenBoundary(ch byte) bool {
	switch ch {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}

func completionScore(candidate string, query string) (int, bool) {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	query = strings.ToLower(strings.TrimSpace(query))
	if candidate == "" {
		return -1, false
	}
	if query == "" {
		return 1, true
	}
	if strings.HasPrefix(candidate, query) {
		return 4, true
	}
	if strings.Contains(candidate, query) {
		return 2, true
	}
	return -1, false
}

func sortSuggestions(items []Suggestion) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].InsertText < items[j].InsertText
	})
}
