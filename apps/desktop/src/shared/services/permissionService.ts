import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import type { PermissionSnapshot } from "@/shared/types/api";

const defaultMenuVisibility: Record<string, PermissionSnapshot["menu_visibility"][string]> = {
  main: "enabled",
  remote_account: "enabled",
  remote_members_roles: "enabled",
  remote_permissions_audit: "enabled",
  workspace_project_config: "enabled",
  workspace_agent: "enabled",
  workspace_model: "enabled",
  workspace_rules: "enabled",
  workspace_skills: "enabled",
  workspace_mcp: "enabled",
  settings_theme: "enabled",
  settings_i18n: "enabled",
  settings_general: "enabled"
};

export async function getPermissionSnapshot(token?: string): Promise<PermissionSnapshot> {
  return withApiFallback(
    "auth.permissionSnapshot",
    () => getControlClient().get<PermissionSnapshot>("/v1/me/permissions", { token }),
    () => createMockPermissionSnapshot()
  );
}

function createMockPermissionSnapshot(): PermissionSnapshot {
  return {
    role: "admin",
    permissions: ["*"],
    menu_visibility: { ...defaultMenuVisibility },
    action_visibility: {
      "project.read": "enabled",
      "project.write": "enabled",
      "conversation.read": "enabled",
      "conversation.write": "enabled",
      "execution.control": "enabled",
      "resource.read": "enabled",
      "resource.write": "enabled",
      "share.request": "enabled",
      "share.approve": "enabled",
      "share.reject": "enabled",
      "share.revoke": "enabled",
      "model_catalog.sync": "enabled",
      "admin.users.manage": "enabled",
      "admin.roles.manage": "enabled",
      "admin.permissions.manage": "enabled",
      "admin.menus.manage": "enabled",
      "admin.policies.manage": "enabled",
      "admin.audit.read": "enabled"
    },
    policy_version: "mock",
    generated_at: new Date().toISOString()
  };
}
