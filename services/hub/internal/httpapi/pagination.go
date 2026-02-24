package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

func parseCursorLimit(r *http.Request) (start int, limit int) {
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if cursor != "" {
		if parsed, err := strconv.Atoi(cursor); err == nil && parsed >= 0 {
			start = parsed
		}
	}

	limit = defaultPageLimit
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	return start, limit
}

func paginateAny(items []any, start int, limit int) ([]any, *string) {
	if start >= len(items) {
		return []any{}, nil
	}

	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	result := make([]any, end-start)
	copy(result, items[start:end])

	if end >= len(items) {
		return result, nil
	}

	cursor := strconv.Itoa(end)
	return result, &cursor
}
