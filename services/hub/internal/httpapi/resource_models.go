package httpapi

type ModelVendorName string

const (
	ModelVendorOpenAI  ModelVendorName = "OpenAI"
	ModelVendorGoogle  ModelVendorName = "Google"
	ModelVendorQwen    ModelVendorName = "Qwen"
	ModelVendorDoubao  ModelVendorName = "Doubao"
	ModelVendorZhipu   ModelVendorName = "Zhipu"
	ModelVendorMiniMax ModelVendorName = "MiniMax"
	ModelVendorLocal   ModelVendorName = "Local"
)

var supportedModelVendors = []ModelVendorName{
	ModelVendorOpenAI,
	ModelVendorGoogle,
	ModelVendorQwen,
	ModelVendorDoubao,
	ModelVendorZhipu,
	ModelVendorMiniMax,
	ModelVendorLocal,
}

type ModelCatalogModel struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type ModelCatalogVendor struct {
	Name   ModelVendorName     `json:"name"`
	Models []ModelCatalogModel `json:"models"`
}

type ModelCatalogResponse struct {
	WorkspaceID string               `json:"workspace_id"`
	Revision    int64                `json:"revision"`
	UpdatedAt   string               `json:"updated_at"`
	Source      string               `json:"source"`
	Vendors     []ModelCatalogVendor `json:"vendors"`
}

type CatalogRootUpdateRequest struct {
	CatalogRoot string `json:"catalog_root"`
}

type CatalogRootResponse struct {
	WorkspaceID string `json:"workspace_id"`
	CatalogRoot string `json:"catalog_root"`
	UpdatedAt   string `json:"updated_at"`
}

type ModelSpec struct {
	Vendor       ModelVendorName `json:"vendor"`
	ModelID      string          `json:"model_id"`
	BaseURL      string          `json:"base_url,omitempty"`
	APIKey       string          `json:"api_key,omitempty"`
	APIKeyMasked string          `json:"api_key_masked,omitempty"`
	TimeoutMS    int             `json:"timeout_ms,omitempty"`
	Params       map[string]any  `json:"params,omitempty"`
}

type RuleSpec struct {
	Content string `json:"content"`
}

type SkillSpec struct {
	Content string `json:"content"`
}

type McpSpec struct {
	Transport       string            `json:"transport"`
	Endpoint        string            `json:"endpoint,omitempty"`
	Command         string            `json:"command,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	Status          string            `json:"status,omitempty"`
	Tools           []string          `json:"tools,omitempty"`
	LastError       string            `json:"last_error,omitempty"`
	LastConnectedAt string            `json:"last_connected_at,omitempty"`
}

type ResourceConfig struct {
	ID          string       `json:"id"`
	WorkspaceID string       `json:"workspace_id"`
	Type        ResourceType `json:"type"`
	Name        string       `json:"name"`
	Enabled     bool         `json:"enabled"`
	Model       *ModelSpec   `json:"model,omitempty"`
	Rule        *RuleSpec    `json:"rule,omitempty"`
	Skill       *SkillSpec   `json:"skill,omitempty"`
	MCP         *McpSpec     `json:"mcp,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type ResourceConfigCreateRequest struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Enabled *bool        `json:"enabled,omitempty"`
	Model   *ModelSpec   `json:"model,omitempty"`
	Rule    *RuleSpec    `json:"rule,omitempty"`
	Skill   *SkillSpec   `json:"skill,omitempty"`
	MCP     *McpSpec     `json:"mcp,omitempty"`
}

type ResourceConfigPatchRequest struct {
	Name    *string    `json:"name,omitempty"`
	Enabled *bool      `json:"enabled,omitempty"`
	Model   *ModelSpec `json:"model,omitempty"`
	Rule    *RuleSpec  `json:"rule,omitempty"`
	Skill   *SkillSpec `json:"skill,omitempty"`
	MCP     *McpSpec   `json:"mcp,omitempty"`
}

type ModelTestResult struct {
	ConfigID  string  `json:"config_id"`
	Status    string  `json:"status"`
	LatencyMS int64   `json:"latency_ms"`
	ErrorCode *string `json:"error_code,omitempty"`
	Message   string  `json:"message"`
	TestedAt  string  `json:"tested_at"`
}

type McpConnectResult struct {
	ConfigID    string   `json:"config_id"`
	Status      string   `json:"status"`
	Tools       []string `json:"tools"`
	ErrorCode   *string  `json:"error_code,omitempty"`
	Message     string   `json:"message"`
	ConnectedAt string   `json:"connected_at"`
}

type ResourceTestLog struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ConfigID    string `json:"config_id"`
	TestType    string `json:"test_type"`
	Result      string `json:"result"`
	LatencyMS   int64  `json:"latency_ms"`
	ErrorCode   string `json:"error_code,omitempty"`
	Details     string `json:"details,omitempty"`
	CreatedAt   string `json:"created_at"`
}
