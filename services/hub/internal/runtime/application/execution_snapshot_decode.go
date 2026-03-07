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
	ModelConfigID            string                                  `json:"model_config_id,omitempty"`
	ModelID                  string                                  `json:"model_id"`
	RuleIDs                  []string                                `json:"rule_ids,omitempty"`
	SkillIDs                 []string                                `json:"skill_ids,omitempty"`
	MCPIDs                   []string                                `json:"mcp_ids,omitempty"`
	ProjectFilePaths         []string                                `json:"project_file_paths,omitempty"`
	RulesDSL                 string                                  `json:"rules_dsl,omitempty"`
	MCPServers               []ExecutionMCPServerSnapshot            `json:"mcp_servers,omitempty"`
	AlwaysLoadedCapabilities []ExecutionCapabilityDescriptorSnapshot `json:"always_loaded_capabilities,omitempty"`
	SearchableCapabilities   []ExecutionCapabilityDescriptorSnapshot `json:"searchable_capabilities,omitempty"`
}

type ExecutionMCPServerSnapshot struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Endpoint  string            `json:"endpoint,omitempty"`
	Command   string            `json:"command,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Tools     []string          `json:"tools,omitempty"`
}

type ExecutionCapabilityDescriptorSnapshot struct {
	ID                  string         `json:"id"`
	Kind                string         `json:"kind"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Source              string         `json:"source"`
	Scope               string         `json:"scope"`
	Version             string         `json:"version"`
	InputSchema         map[string]any `json:"input_schema,omitempty"`
	RiskLevel           string         `json:"risk_level"`
	ReadOnly            bool           `json:"read_only"`
	ConcurrencySafe     bool           `json:"concurrency_safe"`
	RequiresPermissions bool           `json:"requires_permissions"`
	VisibilityPolicy    string         `json:"visibility_policy"`
	PromptBudgetCost    int            `json:"prompt_budget_cost"`
}

type ExecutionCapabilityBudgetsSnapshot struct {
	PromptBudgetChars      int `json:"prompt_budget_chars"`
	SearchThresholdPercent int `json:"search_threshold_percent"`
}

type ExecutionMCPSearchConfigSnapshot struct {
	Enabled     bool `json:"enabled"`
	ResultLimit int  `json:"result_limit"`
}

type ExecutionSubagentDefaultsSnapshot struct {
	MaxTurns     int      `json:"max_turns"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

type ExecutionFeatureFlagsSnapshot struct {
	EnableToolSearch      bool `json:"enable_tool_search"`
	EnableCapabilityGraph bool `json:"enable_capability_graph"`
}

type ExecutionAgentConfigSnapshot struct {
	MaxModelTurns     int                                `json:"max_model_turns"`
	ShowProcessTrace  bool                               `json:"show_process_trace"`
	TraceDetailLevel  string                             `json:"trace_detail_level"`
	DefaultMode       string                             `json:"default_mode,omitempty"`
	BuiltinTools      []string                           `json:"builtin_tools,omitempty"`
	CapabilityBudgets ExecutionCapabilityBudgetsSnapshot `json:"capability_budgets"`
	MCPSearch         ExecutionMCPSearchConfigSnapshot   `json:"mcp_search"`
	OutputStyle       string                             `json:"output_style,omitempty"`
	SubagentDefaults  ExecutionSubagentDefaultsSnapshot  `json:"subagent_defaults"`
	FeatureFlags      ExecutionFeatureFlagsSnapshot      `json:"feature_flags"`
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
	output.MCPServers = cloneExecutionMCPServerSnapshots(input.MCPServers)
	output.AlwaysLoadedCapabilities = cloneExecutionCapabilityDescriptorSnapshots(input.AlwaysLoadedCapabilities)
	output.SearchableCapabilities = cloneExecutionCapabilityDescriptorSnapshots(input.SearchableCapabilities)
	return &output
}

func cloneExecutionAgentConfigSnapshot(input *ExecutionAgentConfigSnapshot) *ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	output := *input
	output.BuiltinTools = append([]string{}, input.BuiltinTools...)
	output.SubagentDefaults = ExecutionSubagentDefaultsSnapshot{
		MaxTurns:     input.SubagentDefaults.MaxTurns,
		AllowedTools: append([]string{}, input.SubagentDefaults.AllowedTools...),
	}
	return &output
}

func cloneExecutionMCPServerSnapshots(input []ExecutionMCPServerSnapshot) []ExecutionMCPServerSnapshot {
	if len(input) == 0 {
		return nil
	}
	output := make([]ExecutionMCPServerSnapshot, 0, len(input))
	for _, item := range input {
		output = append(output, ExecutionMCPServerSnapshot{
			Name:      item.Name,
			Transport: item.Transport,
			Endpoint:  item.Endpoint,
			Command:   item.Command,
			Env:       cloneSnapshotStringMap(item.Env),
			Tools:     append([]string{}, item.Tools...),
		})
	}
	return output
}

func cloneExecutionCapabilityDescriptorSnapshots(input []ExecutionCapabilityDescriptorSnapshot) []ExecutionCapabilityDescriptorSnapshot {
	if len(input) == 0 {
		return nil
	}
	output := make([]ExecutionCapabilityDescriptorSnapshot, 0, len(input))
	for _, item := range input {
		copyItem := item
		copyItem.InputSchema = cloneSnapshotMapAny(item.InputSchema)
		output = append(output, copyItem)
	}
	return output
}

func cloneSnapshotMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneSnapshotStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
