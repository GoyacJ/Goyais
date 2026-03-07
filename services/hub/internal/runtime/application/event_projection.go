package application

import (
	"sort"
	"strings"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

type DiffParser func(payload map[string]any) []runtimedomain.DiffItem

type DiffMerger func(existing []runtimedomain.DiffItem, incoming []runtimedomain.DiffItem) []runtimedomain.DiffItem

type ReplayOptions struct {
	DiffGeneratedType runtimedomain.EventType
	ParseDiff         DiffParser
	MergeDiff         DiffMerger
}

type Projection struct {
	LastSequenceByConversation map[string]int
	DiffsByExecution           map[string][]runtimedomain.DiffItem
}

func ReplayEvents(events []runtimedomain.Event, options ReplayOptions) Projection {
	projection := Projection{
		LastSequenceByConversation: map[string]int{},
		DiffsByExecution:           map[string][]runtimedomain.DiffItem{},
	}
	if len(events) == 0 {
		return projection
	}

	diffGeneratedType := options.DiffGeneratedType
	if diffGeneratedType == "" {
		diffGeneratedType = runtimedomain.EventTypeDiffGenerated
	}

	sorted := append([]runtimedomain.Event{}, events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		if left.ConversationID != right.ConversationID {
			return strings.Compare(left.ConversationID, right.ConversationID) < 0
		}
		if left.Sequence != right.Sequence {
			return left.Sequence < right.Sequence
		}
		if left.ID != right.ID {
			return strings.Compare(left.ID, right.ID) < 0
		}
		return strings.Compare(left.ExecutionID, right.ExecutionID) < 0
	})

	for _, event := range sorted {
		conversationID := strings.TrimSpace(event.ConversationID)
		if conversationID != "" && event.Sequence > projection.LastSequenceByConversation[conversationID] {
			projection.LastSequenceByConversation[conversationID] = event.Sequence
		}

		if event.Type != diffGeneratedType || options.ParseDiff == nil || options.MergeDiff == nil {
			continue
		}
		executionID := strings.TrimSpace(event.ExecutionID)
		if executionID == "" {
			continue
		}
		incoming := options.ParseDiff(event.Payload)
		if len(incoming) == 0 {
			continue
		}
		projection.DiffsByExecution[executionID] = options.MergeDiff(projection.DiffsByExecution[executionID], incoming)
	}

	return projection
}
