package application

import (
	"encoding/json"
	"strings"
)

type ConversationRecordInput struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	Name              string
	QueueState        string
	DefaultMode       string
	ModelConfigID     string
	RuleIDsJSON       string
	SkillIDsJSON      string
	MCPIDsJSON        string
	BaseRevision      int64
	ActiveExecutionID *string
	CreatedAt         string
	UpdatedAt         string
}

type ConversationRecord struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	Name              string
	QueueState        string
	DefaultMode       string
	ModelConfigID     string
	RuleIDs           []string
	SkillIDs          []string
	MCPIDs            []string
	BaseRevision      int64
	ActiveExecutionID *string
	CreatedAt         string
	UpdatedAt         string
}

type ConversationSnapshotInspector struct {
	Tab string `json:"tab"`
}

type ConversationSnapshotMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"session_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
	QueueIndex     *int   `json:"queue_index,omitempty"`
	CanRollback    *bool  `json:"can_rollback,omitempty"`
}

type ConversationSnapshotRecordInput struct {
	ID                     string
	ConversationID         string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorStateJSON     string
	MessagesJSON           string
	ExecutionIDsJSON       string
	CreatedAt              string
}

type ConversationSnapshotRecord struct {
	ID                     string
	ConversationID         string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorState         ConversationSnapshotInspector
	Messages               []ConversationSnapshotMessage
	ExecutionIDs           []string
	CreatedAt              string
}

type ConversationSnapshotWriteInput struct {
	ID                     string
	ConversationID         string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorState         ConversationSnapshotInspector
	Messages               []ConversationSnapshotMessage
	ExecutionIDs           []string
	CreatedAt              string
}

type ConversationSnapshotWriteRecord struct {
	ID                     string
	ConversationID         string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorState         ConversationSnapshotInspector
	Messages               []ConversationSnapshotMessage
	ExecutionIDs           []string
	CreatedAt              string
}

type ConversationWriteInput struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	Name              string
	QueueState        string
	DefaultMode       string
	ModelConfigID     string
	RuleIDs           []string
	SkillIDs          []string
	MCPIDs            []string
	BaseRevision      int64
	ActiveExecutionID *string
	CreatedAt         string
	UpdatedAt         string
}

type ConversationWriteRecord struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	Name              string
	QueueState        string
	DefaultMode       string
	ModelConfigID     string
	RuleIDs           []string
	SkillIDs          []string
	MCPIDs            []string
	BaseRevision      int64
	ActiveExecutionID *string
	CreatedAt         string
	UpdatedAt         string
}

func ParseConversationRecords(inputs []ConversationRecordInput) ([]ConversationRecord, error) {
	if len(inputs) == 0 {
		return []ConversationRecord{}, nil
	}

	records := make([]ConversationRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ConversationRecord{
			ID:                input.ID,
			WorkspaceID:       input.WorkspaceID,
			ProjectID:         input.ProjectID,
			Name:              input.Name,
			QueueState:        strings.TrimSpace(input.QueueState),
			DefaultMode:       strings.TrimSpace(input.DefaultMode),
			ModelConfigID:     input.ModelConfigID,
			RuleIDs:           []string{},
			SkillIDs:          []string{},
			MCPIDs:            []string{},
			BaseRevision:      input.BaseRevision,
			ActiveExecutionID: input.ActiveExecutionID,
			CreatedAt:         input.CreatedAt,
			UpdatedAt:         input.UpdatedAt,
		}

		if strings.TrimSpace(input.RuleIDsJSON) != "" {
			if err := json.Unmarshal([]byte(input.RuleIDsJSON), &record.RuleIDs); err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(input.SkillIDsJSON) != "" {
			if err := json.Unmarshal([]byte(input.SkillIDsJSON), &record.SkillIDs); err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(input.MCPIDsJSON) != "" {
			if err := json.Unmarshal([]byte(input.MCPIDsJSON), &record.MCPIDs); err != nil {
				return nil, err
			}
		}

		if record.ActiveExecutionID != nil && strings.TrimSpace(*record.ActiveExecutionID) == "" {
			record.ActiveExecutionID = nil
		}
		records = append(records, record)
	}

	return records, nil
}

func ParseConversationSnapshotRecords(inputs []ConversationSnapshotRecordInput) ([]ConversationSnapshotRecord, error) {
	if len(inputs) == 0 {
		return []ConversationSnapshotRecord{}, nil
	}

	records := make([]ConversationSnapshotRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ConversationSnapshotRecord{
			ID:                     input.ID,
			ConversationID:         input.ConversationID,
			RollbackPointMessageID: input.RollbackPointMessageID,
			QueueState:             strings.TrimSpace(input.QueueState),
			CreatedAt:              input.CreatedAt,
		}

		if input.WorktreeRef != nil && strings.TrimSpace(*input.WorktreeRef) != "" {
			value := strings.TrimSpace(*input.WorktreeRef)
			record.WorktreeRef = &value
		}
		if strings.TrimSpace(input.InspectorStateJSON) != "" {
			if err := json.Unmarshal([]byte(input.InspectorStateJSON), &record.InspectorState); err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(input.MessagesJSON) != "" {
			if err := json.Unmarshal([]byte(input.MessagesJSON), &record.Messages); err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(input.ExecutionIDsJSON) != "" {
			if err := json.Unmarshal([]byte(input.ExecutionIDsJSON), &record.ExecutionIDs); err != nil {
				return nil, err
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func NormalizeConversationSnapshotWriteRecords(inputs []ConversationSnapshotWriteInput) []ConversationSnapshotWriteRecord {
	if len(inputs) == 0 {
		return []ConversationSnapshotWriteRecord{}
	}

	records := make([]ConversationSnapshotWriteRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ConversationSnapshotWriteRecord{
			ID:                     input.ID,
			ConversationID:         input.ConversationID,
			RollbackPointMessageID: input.RollbackPointMessageID,
			QueueState:             strings.TrimSpace(input.QueueState),
			InspectorState:         input.InspectorState,
			ExecutionIDs:           append([]string{}, input.ExecutionIDs...),
			CreatedAt:              input.CreatedAt,
		}
		if input.WorktreeRef != nil && strings.TrimSpace(*input.WorktreeRef) != "" {
			value := strings.TrimSpace(*input.WorktreeRef)
			record.WorktreeRef = &value
		}
		if len(input.Messages) > 0 {
			record.Messages = make([]ConversationSnapshotMessage, 0, len(input.Messages))
			for _, message := range input.Messages {
				record.Messages = append(record.Messages, ConversationSnapshotMessage{
					ID:             message.ID,
					ConversationID: message.ConversationID,
					Role:           strings.TrimSpace(message.Role),
					Content:        message.Content,
					CreatedAt:      message.CreatedAt,
					QueueIndex:     cloneMessageOptionalInt(message.QueueIndex),
					CanRollback:    cloneMessageOptionalBool(message.CanRollback),
				})
			}
		}
		records = append(records, record)
	}
	return records
}

func NormalizeConversationWriteRecords(inputs []ConversationWriteInput) []ConversationWriteRecord {
	if len(inputs) == 0 {
		return []ConversationWriteRecord{}
	}

	records := make([]ConversationWriteRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ConversationWriteRecord{
			ID:            input.ID,
			WorkspaceID:   input.WorkspaceID,
			ProjectID:     input.ProjectID,
			Name:          input.Name,
			QueueState:    strings.TrimSpace(input.QueueState),
			DefaultMode:   strings.TrimSpace(input.DefaultMode),
			ModelConfigID: input.ModelConfigID,
			RuleIDs:       normalizeUniqueTrimmedStringList(input.RuleIDs),
			SkillIDs:      normalizeUniqueTrimmedStringList(input.SkillIDs),
			MCPIDs:        normalizeUniqueTrimmedStringList(input.MCPIDs),
			BaseRevision:  input.BaseRevision,
			CreatedAt:     input.CreatedAt,
			UpdatedAt:     input.UpdatedAt,
		}
		if input.ActiveExecutionID != nil && strings.TrimSpace(*input.ActiveExecutionID) != "" {
			value := strings.TrimSpace(*input.ActiveExecutionID)
			record.ActiveExecutionID = &value
		}
		records = append(records, record)
	}
	return records
}

func normalizeUniqueTrimmedStringList(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
