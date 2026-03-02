package application

import (
	"encoding/json"
	"strings"
)

type ExecutionModelRuntime struct {
	RequestTimeoutMS *int `json:"request_timeout_ms,omitempty"`
}

type ExecutionModelSnapshot struct {
	ConfigID   string                 `json:"config_id,omitempty"`
	Vendor     string                 `json:"vendor,omitempty"`
	ModelID    string                 `json:"model_id"`
	BaseURL    string                 `json:"base_url,omitempty"`
	BaseURLKey string                 `json:"base_url_key,omitempty"`
	Runtime    *ExecutionModelRuntime `json:"runtime,omitempty"`
	Params     map[string]any         `json:"params,omitempty"`
}

type ExecutionResourceProfileSnapshot struct {
	ModelConfigID    string   `json:"model_config_id,omitempty"`
	ModelID          string   `json:"model_id"`
	RuleIDs          []string `json:"rule_ids,omitempty"`
	SkillIDs         []string `json:"skill_ids,omitempty"`
	MCPIDs           []string `json:"mcp_ids,omitempty"`
	ProjectFilePaths []string `json:"project_file_paths,omitempty"`
}

type ExecutionAgentConfigSnapshot struct {
	MaxModelTurns    int    `json:"max_model_turns"`
	ShowProcessTrace bool   `json:"show_process_trace"`
	TraceDetailLevel string `json:"trace_detail_level"`
}

type ExecutionRecordInput struct {
	ID                          string
	WorkspaceID                 string
	ConversationID              string
	MessageID                   string
	State                       string
	Mode                        string
	ModelID                     string
	ModeSnapshot                string
	ModelSnapshotJSON           string
	ResourceProfileSnapshotJSON *string
	AgentConfigSnapshotJSON     *string
	TokensIn                    int
	TokensOut                   int
	ProjectRevisionSnapshot     int64
	QueueIndex                  int
	TraceID                     string
	CreatedAt                   string
	UpdatedAt                   string
}

type ExecutionRecord struct {
	ID                      string
	WorkspaceID             string
	ConversationID          string
	MessageID               string
	State                   string
	Mode                    string
	ModelID                 string
	ModeSnapshot            string
	ModelSnapshot           ExecutionModelSnapshot
	ResourceProfileSnapshot *ExecutionResourceProfileSnapshot
	AgentConfigSnapshot     *ExecutionAgentConfigSnapshot
	TokensIn                int
	TokensOut               int
	ProjectRevisionSnapshot int64
	QueueIndex              int
	TraceID                 string
	CreatedAt               string
	UpdatedAt               string
}

type ExecutionWriteInput struct {
	ID                      string
	WorkspaceID             string
	ConversationID          string
	MessageID               string
	State                   string
	Mode                    string
	ModelID                 string
	ModeSnapshot            string
	ModelSnapshot           ExecutionModelSnapshot
	ResourceProfileSnapshot *ExecutionResourceProfileSnapshot
	AgentConfigSnapshot     *ExecutionAgentConfigSnapshot
	TokensIn                int
	TokensOut               int
	ProjectRevisionSnapshot int64
	QueueIndex              int
	TraceID                 string
	CreatedAt               string
	UpdatedAt               string
}

type ExecutionWriteRecord struct {
	ID                      string
	WorkspaceID             string
	ConversationID          string
	MessageID               string
	State                   string
	Mode                    string
	ModelID                 string
	ModeSnapshot            string
	ModelSnapshot           ExecutionModelSnapshot
	ResourceProfileSnapshot *ExecutionResourceProfileSnapshot
	AgentConfigSnapshot     *ExecutionAgentConfigSnapshot
	TokensIn                int
	TokensOut               int
	ProjectRevisionSnapshot int64
	QueueIndex              int
	TraceID                 string
	CreatedAt               string
	UpdatedAt               string
}

func NormalizeExecutionWriteRecords(inputs []ExecutionWriteInput) []ExecutionWriteRecord {
	if len(inputs) == 0 {
		return []ExecutionWriteRecord{}
	}

	records := make([]ExecutionWriteRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ExecutionWriteRecord{
			ID:                      input.ID,
			WorkspaceID:             input.WorkspaceID,
			ConversationID:          input.ConversationID,
			MessageID:               input.MessageID,
			State:                   strings.TrimSpace(input.State),
			Mode:                    strings.TrimSpace(input.Mode),
			ModelID:                 input.ModelID,
			ModeSnapshot:            strings.TrimSpace(input.ModeSnapshot),
			ModelSnapshot:           cloneExecutionModelSnapshot(input.ModelSnapshot),
			ResourceProfileSnapshot: cloneExecutionResourceProfileSnapshot(input.ResourceProfileSnapshot),
			AgentConfigSnapshot:     cloneExecutionAgentConfigSnapshot(input.AgentConfigSnapshot),
			TokensIn:                input.TokensIn,
			TokensOut:               input.TokensOut,
			ProjectRevisionSnapshot: input.ProjectRevisionSnapshot,
			QueueIndex:              input.QueueIndex,
			TraceID:                 input.TraceID,
			CreatedAt:               input.CreatedAt,
			UpdatedAt:               input.UpdatedAt,
		}
		records = append(records, record)
	}

	return records
}

func ParseExecutionRecords(inputs []ExecutionRecordInput) ([]ExecutionRecord, error) {
	if len(inputs) == 0 {
		return []ExecutionRecord{}, nil
	}

	records := make([]ExecutionRecord, 0, len(inputs))
	for _, input := range inputs {
		record := ExecutionRecord{
			ID:                      input.ID,
			WorkspaceID:             input.WorkspaceID,
			ConversationID:          input.ConversationID,
			MessageID:               input.MessageID,
			State:                   strings.TrimSpace(input.State),
			Mode:                    strings.TrimSpace(input.Mode),
			ModelID:                 input.ModelID,
			ModeSnapshot:            strings.TrimSpace(input.ModeSnapshot),
			TokensIn:                input.TokensIn,
			TokensOut:               input.TokensOut,
			ProjectRevisionSnapshot: input.ProjectRevisionSnapshot,
			QueueIndex:              input.QueueIndex,
			TraceID:                 input.TraceID,
			CreatedAt:               input.CreatedAt,
			UpdatedAt:               input.UpdatedAt,
		}

		if strings.TrimSpace(input.ModelSnapshotJSON) != "" {
			if err := json.Unmarshal([]byte(input.ModelSnapshotJSON), &record.ModelSnapshot); err != nil {
				return nil, err
			}
			legacyModelSnapshot := struct {
				TimeoutMS *int `json:"timeout_ms"`
			}{}
			if err := json.Unmarshal([]byte(input.ModelSnapshotJSON), &legacyModelSnapshot); err != nil {
				return nil, err
			}
			if (record.ModelSnapshot.Runtime == nil || record.ModelSnapshot.Runtime.RequestTimeoutMS == nil) && legacyModelSnapshot.TimeoutMS != nil {
				value := *legacyModelSnapshot.TimeoutMS
				record.ModelSnapshot.Runtime = &ExecutionModelRuntime{RequestTimeoutMS: &value}
			}
		}

		if input.ResourceProfileSnapshotJSON != nil && strings.TrimSpace(*input.ResourceProfileSnapshotJSON) != "" {
			resourceSnapshot := ExecutionResourceProfileSnapshot{}
			if err := json.Unmarshal([]byte(*input.ResourceProfileSnapshotJSON), &resourceSnapshot); err != nil {
				return nil, err
			}
			record.ResourceProfileSnapshot = &resourceSnapshot
		}

		if input.AgentConfigSnapshotJSON != nil && strings.TrimSpace(*input.AgentConfigSnapshotJSON) != "" {
			configSnapshot := ExecutionAgentConfigSnapshot{}
			if err := json.Unmarshal([]byte(*input.AgentConfigSnapshotJSON), &configSnapshot); err != nil {
				return nil, err
			}
			record.AgentConfigSnapshot = &configSnapshot
		}

		records = append(records, record)
	}

	return records, nil
}

func cloneExecutionModelSnapshot(input ExecutionModelSnapshot) ExecutionModelSnapshot {
	output := input
	if input.Runtime != nil {
		value := *input.Runtime
		if input.Runtime.RequestTimeoutMS != nil {
			timeout := *input.Runtime.RequestTimeoutMS
			value.RequestTimeoutMS = &timeout
		}
		output.Runtime = &value
	}
	if input.Params != nil {
		output.Params = make(map[string]any, len(input.Params))
		for key, value := range input.Params {
			output.Params[key] = value
		}
	}
	return output
}

func cloneExecutionResourceProfileSnapshot(input *ExecutionResourceProfileSnapshot) *ExecutionResourceProfileSnapshot {
	if input == nil {
		return nil
	}
	output := *input
	output.RuleIDs = append([]string{}, input.RuleIDs...)
	output.SkillIDs = append([]string{}, input.SkillIDs...)
	output.MCPIDs = append([]string{}, input.MCPIDs...)
	output.ProjectFilePaths = append([]string{}, input.ProjectFilePaths...)
	return &output
}

func cloneExecutionAgentConfigSnapshot(input *ExecutionAgentConfigSnapshot) *ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	output := *input
	return &output
}
