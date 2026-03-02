package application

import "strings"

type ConversationMessageRecordInput struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	QueueIndex     *int
	CanRollback    *bool
	CreatedAt      string
}

type ConversationMessageRecord struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	QueueIndex     *int
	CanRollback    *bool
	CreatedAt      string
}

func ParseConversationMessageRecords(inputs []ConversationMessageRecordInput) ([]ConversationMessageRecord, error) {
	if len(inputs) == 0 {
		return []ConversationMessageRecord{}, nil
	}

	records := make([]ConversationMessageRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ConversationMessageRecord{
			ID:             input.ID,
			ConversationID: input.ConversationID,
			Role:           strings.TrimSpace(input.Role),
			Content:        input.Content,
			QueueIndex:     cloneMessageOptionalInt(input.QueueIndex),
			CanRollback:    cloneMessageOptionalBool(input.CanRollback),
			CreatedAt:      input.CreatedAt,
		}
		records = append(records, record)
	}

	return records, nil
}

func cloneMessageOptionalInt(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneMessageOptionalBool(input *bool) *bool {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}
