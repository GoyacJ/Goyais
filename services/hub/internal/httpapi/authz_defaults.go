package httpapi

type adminPermission struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type adminMenu struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

func defaultMenuConfigs() []adminMenu {
	return []adminMenu{
		{Key: "main", Label: "主界面", Enabled: true},
		{Key: "remote_account", Label: "账号信息", Enabled: true},
		{Key: "remote_members_roles", Label: "成员与角色", Enabled: true},
		{Key: "remote_permissions_audit", Label: "权限与审计", Enabled: true},
		{Key: "workspace_project_config", Label: "项目配置", Enabled: true},
		{Key: "workspace_agent", Label: "Agent配置", Enabled: true},
		{Key: "workspace_model", Label: "模型配置", Enabled: true},
		{Key: "workspace_rules", Label: "规则配置", Enabled: true},
		{Key: "workspace_skills", Label: "技能配置", Enabled: true},
		{Key: "workspace_mcp", Label: "MCP配置", Enabled: true},
		{Key: "settings_theme", Label: "主题", Enabled: true},
		{Key: "settings_i18n", Label: "国际化", Enabled: true},
		{Key: "settings_general", Label: "通用设置", Enabled: true},
	}
}

func defaultMenuVisibility(role Role) map[string]PermissionVisibility {
	visibility := map[string]PermissionVisibility{
		"main":                     PermissionVisibilityEnabled,
		"remote_account":           PermissionVisibilityEnabled,
		"remote_members_roles":     PermissionVisibilityHidden,
		"remote_permissions_audit": PermissionVisibilityHidden,
		"workspace_project_config": PermissionVisibilityReadonly,
		"workspace_agent":          PermissionVisibilityReadonly,
		"workspace_model":          PermissionVisibilityReadonly,
		"workspace_rules":          PermissionVisibilityReadonly,
		"workspace_skills":         PermissionVisibilityReadonly,
		"workspace_mcp":            PermissionVisibilityReadonly,
		"settings_theme":           PermissionVisibilityEnabled,
		"settings_i18n":            PermissionVisibilityEnabled,
		"settings_general":         PermissionVisibilityEnabled,
	}

	switch role {
	case RoleDeveloper:
		visibility["workspace_project_config"] = PermissionVisibilityEnabled
		visibility["workspace_agent"] = PermissionVisibilityEnabled
		visibility["workspace_model"] = PermissionVisibilityEnabled
		visibility["workspace_rules"] = PermissionVisibilityEnabled
		visibility["workspace_skills"] = PermissionVisibilityEnabled
		visibility["workspace_mcp"] = PermissionVisibilityEnabled
	case RoleApprover:
		visibility["workspace_project_config"] = PermissionVisibilityEnabled
		visibility["workspace_agent"] = PermissionVisibilityEnabled
		visibility["workspace_model"] = PermissionVisibilityEnabled
		visibility["workspace_rules"] = PermissionVisibilityEnabled
		visibility["workspace_skills"] = PermissionVisibilityEnabled
		visibility["workspace_mcp"] = PermissionVisibilityEnabled
		visibility["remote_permissions_audit"] = PermissionVisibilityEnabled
	case RoleAdmin:
		for menuKey := range visibility {
			visibility[menuKey] = PermissionVisibilityEnabled
		}
	}

	return visibility
}

func defaultABACPolicies(workspaceID string) []ABACPolicy {
	return []ABACPolicy{
		{
			ID:          "abac_" + workspaceID + "_allow_self_workspace",
			WorkspaceID: workspaceID,
			Name:        "allow self workspace",
			Effect:      ABACEffectAllow,
			Priority:    100,
			Enabled:     true,
			SubjectExpr: map[string]any{
				"roles": map[string]any{
					"in": []any{string(RoleDeveloper), string(RoleApprover), string(RoleAdmin)},
				},
			},
			ResourceExpr: map[string]any{
				"workspace_id": map[string]any{"eq": "$subject.workspace_id"},
			},
			ActionExpr: map[string]any{
				"name": map[string]any{
					"in": []any{
						"project.read", "project.write", "conversation.read", "conversation.write", "execution.control",
						"resource.read", "resource.write", "resource_config.read", "resource_config.write", "resource_config.delete",
						"project_config.read", "model.test", "mcp.connect", "catalog.update_root",
						"share.request", "share.revoke", "admin.audit.read",
					},
				},
			},
			ContextExpr: map[string]any{},
		},
		{
			ID:          "abac_" + workspaceID + "_share_approval",
			WorkspaceID: workspaceID,
			Name:        "allow share approve/reject",
			Effect:      ABACEffectAllow,
			Priority:    110,
			Enabled:     true,
			SubjectExpr: map[string]any{
				"roles": map[string]any{
					"in": []any{string(RoleApprover), string(RoleAdmin)},
				},
			},
			ResourceExpr: map[string]any{
				"workspace_id": map[string]any{"eq": "$subject.workspace_id"},
			},
			ActionExpr: map[string]any{
				"name": map[string]any{"in": []any{"share.approve", "share.reject"}},
			},
			ContextExpr: map[string]any{
				"risk_level": map[string]any{"in": []any{"high", "critical"}},
			},
		},
		{
			ID:          "abac_" + workspaceID + "_admin_manage",
			WorkspaceID: workspaceID,
			Name:        "allow admin manage",
			Effect:      ABACEffectAllow,
			Priority:    120,
			Enabled:     true,
			SubjectExpr: map[string]any{
				"roles": map[string]any{"contains": string(RoleAdmin)},
			},
			ResourceExpr: map[string]any{
				"workspace_id": map[string]any{"eq": "$subject.workspace_id"},
			},
			ActionExpr: map[string]any{
				"name": map[string]any{
					"in": []any{
						"admin.users.manage", "admin.roles.manage", "admin.permissions.manage",
						"admin.menus.manage", "admin.policies.manage",
					},
				},
			},
			ContextExpr: map[string]any{},
		},
		{
			ID:          "abac_" + workspaceID + "_deny_non_admin_high_risk_admin_manage",
			WorkspaceID: workspaceID,
			Name:        "deny non-admin high risk admin manage",
			Effect:      ABACEffectDeny,
			Priority:    1000,
			Enabled:     true,
			SubjectExpr: map[string]any{
				"roles": map[string]any{
					"neq": string(RoleAdmin),
				},
			},
			ResourceExpr: map[string]any{
				"workspace_id": map[string]any{"eq": "$subject.workspace_id"},
			},
			ActionExpr: map[string]any{
				"name": map[string]any{
					"in": []any{
						"admin.users.manage", "admin.roles.manage", "admin.permissions.manage",
						"admin.menus.manage", "admin.policies.manage",
					},
				},
			},
			ContextExpr: map[string]any{
				"risk_level": map[string]any{"in": []any{"high", "critical"}},
			},
		},
	}
}
