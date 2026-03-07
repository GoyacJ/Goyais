package application

import (
	"sort"
	"strings"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

type ExecutionEventReadModel struct {
	OrderedEvents              []runtimedomain.Event
	EventsByConversation       map[string][]runtimedomain.Event
	LastSequenceByConversation map[string]int
	DiffsByExecution           map[string][]runtimedomain.DiffItem
}

func BuildExecutionEventReadModel(events []runtimedomain.Event, options ReplayOptions) ExecutionEventReadModel {
	model := ExecutionEventReadModel{
		OrderedEvents:              []runtimedomain.Event{},
		EventsByConversation:       map[string][]runtimedomain.Event{},
		LastSequenceByConversation: map[string]int{},
		DiffsByExecution:           map[string][]runtimedomain.DiffItem{},
	}
	if len(events) == 0 {
		return model
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
	model.OrderedEvents = sorted
	for _, event := range sorted {
		model.EventsByConversation[event.ConversationID] = append(model.EventsByConversation[event.ConversationID], event)
	}

	projection := ReplayEvents(sorted, options)
	model.LastSequenceByConversation = projection.LastSequenceByConversation
	model.DiffsByExecution = projection.DiffsByExecution
	return model
}
