package domain

import "context"

type ResourceType string

const (
	ResourceTypeModel ResourceType = "model"
	ResourceTypeRule  ResourceType = "rule"
	ResourceTypeSkill ResourceType = "skill"
	ResourceTypeMCP   ResourceType = "mcp"
)

type ModelSpec struct {
	Vendor         string
	ModelID        string
	BaseURL        string
	BaseURLKey     string
	APIKey         string
	APIKeyMasked   string
	TokenThreshold *int
	Runtime        *ModelRuntimeSpec
	Params         map[string]any
}

type ModelRuntimeSpec struct {
	RequestTimeoutMS *int
}

type RuleSpec struct {
	Content string
}

type SkillSpec struct {
	Content string
}

type MCPConfig struct {
	Transport       string
	Endpoint        string
	Command         string
	Env             map[string]string
	Status          string
	Tools           []string
	LastError       string
	LastConnectedAt string
}

type ResourceConfig struct {
	ID          string
	WorkspaceID WorkspaceID
	Type        ResourceType
	Name        string
	Enabled     bool
	Version     int
	IsDeleted   bool
	DeletedAt   *string
	Model       *ModelSpec
	Rule        *RuleSpec
	Skill       *SkillSpec
	MCP         *MCPConfig
	TokensInTotal  int
	TokensOutTotal int
	TokensTotal    int
	CreatedAt   string
	UpdatedAt   string
}

type ProjectResourceConfig struct {
	ProjectID            string
	ModelConfigIDs       []string
	DefaultModelConfigID *string
	TokenThreshold       *int
	ModelTokenThresholds map[string]int
	RuleIDs              []string
	SkillIDs             []string
	MCPIDs               []string
	UpdatedAt            string
}

type SessionResourceSnapshot struct {
	SessionID          SessionID
	ResourceConfigID   string
	ResourceType       ResourceType
	ResourceVersion    int
	IsDeprecated       bool
	FallbackResourceID *string
	SnapshotAt         string
	CapturedConfig     ResourceConfig
}

type SessionResourceState struct {
	SessionID     SessionID
	WorkspaceID   WorkspaceID
	ProjectID     string
	ModelConfigID string
	RuleIDs       []string
	SkillIDs      []string
	MCPIDs        []string
	UpdatedAt     string
}

type ResourceConfigRepository interface {
	GetResourceConfig(ctx context.Context, workspaceID WorkspaceID, configID string) (ResourceConfig, bool, error)
	ListSessionResourceSnapshots(ctx context.Context, sessionID SessionID) ([]SessionResourceSnapshot, error)
}
