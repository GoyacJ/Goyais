package application

import (
	"strings"
	"time"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

type AppendOptions struct {
	Event             runtimedomain.Event
	CurrentSequence   int
	ExistingDiffs     []runtimedomain.DiffItem
	Now               time.Time
	GenerateEventID   func() string
	GenerateTraceID   func() string
	DiffGeneratedType runtimedomain.EventType
	ParseDiff         DiffParser
	MergeDiff         DiffMerger
}

type AppendResult struct {
	Event       runtimedomain.Event
	UpdatedDiff []runtimedomain.DiffItem
}

func NormalizeAppendedEvent(options AppendOptions) AppendResult {
	event := options.Event
	if strings.TrimSpace(event.ID) == "" && options.GenerateEventID != nil {
		event.ID = options.GenerateEventID()
	}
	if strings.TrimSpace(event.TraceID) == "" && options.GenerateTraceID != nil {
		event.TraceID = options.GenerateTraceID()
	}
	if strings.TrimSpace(event.Timestamp) == "" {
		now := options.Now
		if now.IsZero() {
			now = time.Now().UTC()
		} else {
			now = now.UTC()
		}
		event.Timestamp = now.Format(time.RFC3339)
	}
	if event.Sequence <= 0 {
		event.Sequence = options.CurrentSequence + 1
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}

	updatedDiff := append([]runtimedomain.DiffItem{}, options.ExistingDiffs...)
	diffGeneratedType := options.DiffGeneratedType
	if diffGeneratedType == "" {
		diffGeneratedType = runtimedomain.EventTypeDiffGenerated
	}
	if event.Type == diffGeneratedType && strings.TrimSpace(event.ExecutionID) != "" && options.ParseDiff != nil && options.MergeDiff != nil {
		incoming := options.ParseDiff(event.Payload)
		if len(incoming) > 0 {
			updatedDiff = options.MergeDiff(options.ExistingDiffs, incoming)
		}
	}

	return AppendResult{
		Event:       event,
		UpdatedDiff: updatedDiff,
	}
}
