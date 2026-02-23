import type { MenuKey } from "@/shared/types/api";

export type MenuContext = "account" | "settings";

export type MenuSchemaEntry = {
  key: MenuKey;
  labelKey: string;
  path: string;
};

export const menuSchema: MenuSchemaEntry[] = [
  { key: "main", labelKey: "menu.main", path: "/main" },
  { key: "remote_account", labelKey: "menu.remoteAccount", path: "/remote/account" },
  { key: "remote_members_roles", labelKey: "menu.remoteMembersRoles", path: "/remote/members-roles" },
  { key: "remote_permissions_audit", labelKey: "menu.remotePermissionsAudit", path: "/remote/permissions-audit" },
  { key: "workspace_project_config", labelKey: "menu.workspaceProjectConfig", path: "/workspace/project-config" },
  { key: "workspace_agent", labelKey: "menu.workspaceAgent", path: "/workspace/agent" },
  { key: "workspace_model", labelKey: "menu.workspaceModel", path: "/workspace/model" },
  { key: "workspace_rules", labelKey: "menu.workspaceRules", path: "/workspace/rules" },
  { key: "workspace_skills", labelKey: "menu.workspaceSkills", path: "/workspace/skills" },
  { key: "workspace_mcp", labelKey: "menu.workspaceMcp", path: "/workspace/mcp" },
  { key: "settings_theme", labelKey: "menu.settingsTheme", path: "/settings/theme" },
  { key: "settings_i18n", labelKey: "menu.settingsI18n", path: "/settings/i18n" },
  { key: "settings_general", labelKey: "menu.settingsGeneral", path: "/settings/general" }
];

const accountMenuKeys: MenuKey[] = [
  "remote_account",
  "remote_members_roles",
  "remote_permissions_audit",
  "workspace_project_config",
  "workspace_agent",
  "workspace_model",
  "workspace_rules",
  "workspace_skills",
  "workspace_mcp"
];

const settingsMenuKeys: MenuKey[] = [
  "workspace_agent",
  "workspace_model",
  "workspace_rules",
  "workspace_skills",
  "workspace_mcp",
  "workspace_project_config",
  "settings_theme",
  "settings_i18n",
  "settings_general"
];

export function getMenuKeysForContext(context: MenuContext): MenuKey[] {
  return context === "account" ? [...accountMenuKeys] : [...settingsMenuKeys];
}

export function getMenuSchemaMap(): Map<MenuKey, MenuSchemaEntry> {
  return new Map(menuSchema.map((entry) => [entry.key, entry]));
}
