package application

import (
	"strings"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func AppendEventWithHistoryLimit(existing []runtimedomain.Event, event runtimedomain.Event, maxHistory int) []runtimedomain.Event {
	updated := append(append([]runtimedomain.Event{}, existing...), event)
	if maxHistory <= 0 || len(updated) <= maxHistory {
		return updated
	}
	start := len(updated) - maxHistory
	return append([]runtimedomain.Event{}, updated[start:]...)
}

func ListEventsSince(events []runtimedomain.Event, lastEventID string) ([]runtimedomain.Event, bool) {
	if len(events) == 0 {
		return []runtimedomain.Event{}, strings.TrimSpace(lastEventID) != ""
	}
	if strings.TrimSpace(lastEventID) == "" {
		result := make([]runtimedomain.Event, len(events))
		copy(result, events)
		return result, false
	}

	start := 0
	found := false
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].ID == lastEventID {
			start = index + 1
			found = true
			break
		}
	}
	if !found {
		result := make([]runtimedomain.Event, len(events))
		copy(result, events)
		return result, true
	}
	if start >= len(events) {
		return []runtimedomain.Event{}, false
	}
	result := make([]runtimedomain.Event, len(events)-start)
	copy(result, events[start:])
	return result, false
}
