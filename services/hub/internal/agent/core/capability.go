// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

// CapabilityKind identifies one capability family in Tooling V2.
type CapabilityKind string

const (
	CapabilityKindBuiltinTool CapabilityKind = "builtin_tool"
	CapabilityKindMCPTool     CapabilityKind = "mcp_tool"
	CapabilityKindMCPPrompt   CapabilityKind = "mcp_prompt"
	CapabilityKindSkill       CapabilityKind = "skill"
	CapabilityKindSlash       CapabilityKind = "slash_command"
	CapabilityKindSubagent    CapabilityKind = "subagent"
	CapabilityKindOutputStyle CapabilityKind = "output_style"
)

// CapabilityScope describes where a capability originates from.
type CapabilityScope string

const (
	CapabilityScopeSystem    CapabilityScope = "system"
	CapabilityScopeWorkspace CapabilityScope = "workspace"
	CapabilityScopeProject   CapabilityScope = "project"
	CapabilityScopeUser      CapabilityScope = "user"
	CapabilityScopeLocal     CapabilityScope = "local"
	CapabilityScopePlugin    CapabilityScope = "plugin"
	CapabilityScopeManaged   CapabilityScope = "managed"
)

// CapabilityVisibilityPolicy controls whether a capability is prompt-exposed
// immediately or only discoverable through search.
type CapabilityVisibilityPolicy string

const (
	CapabilityVisibilityAlwaysLoaded CapabilityVisibilityPolicy = "always_loaded"
	CapabilityVisibilitySearchable   CapabilityVisibilityPolicy = "searchable"
)

// CapabilityDescriptor is the unified Tooling V2 capability declaration.
type CapabilityDescriptor struct {
	ID                  string
	Kind                CapabilityKind
	Name                string
	Description         string
	Source              string
	Scope               CapabilityScope
	Version             string
	InputSchema         map[string]any
	RiskLevel           string
	ReadOnly            bool
	ConcurrencySafe     bool
	RequiresPermissions bool
	VisibilityPolicy    CapabilityVisibilityPolicy
	PromptBudgetCost    int
}

// MCPServerConfig is the strong runtime snapshot for one MCP server.
type MCPServerConfig struct {
	Name      string
	Transport string
	Endpoint  string
	Command   string
	Env       map[string]string
	Tools     []string
}

// RuntimeModelConfig is the strong runtime snapshot for one model request.
type RuntimeModelConfig struct {
	ProviderName  string
	Endpoint      string
	ModelName     string
	APIKey        string
	Params        map[string]any
	TimeoutMS     int
	MaxModelTurns int
}

// RuntimeToolingConfig is the strong runtime snapshot for the Tooling V2
// surface seen by one run.
type RuntimeToolingConfig struct {
	PermissionMode          PermissionMode
	RulesDSL                string
	MCPServers              []MCPServerConfig
	AlwaysLoadedCapabilities []CapabilityDescriptor
	SearchableCapabilities  []CapabilityDescriptor
	PromptBudgetChars       int
	MCPSearchEnabled        bool
	SearchThresholdRatio    float64
}

// RuntimeConfig bundles the resolved model and tooling runtime snapshots.
type RuntimeConfig struct {
	Model   RuntimeModelConfig
	Tooling RuntimeToolingConfig
}
