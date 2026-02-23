import { reactive } from "vue";

import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { MenuKey, MenuVisibility, PermissionVisibility } from "@/shared/types/api";

type NavigationState = {
  visibility: MenuVisibility;
};

const allEnabled: MenuVisibility = {
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
  settings_updates_diagnostics: "enabled",
  settings_general: "enabled"
};

export const navigationStore = reactive<NavigationState>({
  visibility: { ...allEnabled }
});

export function resetNavigationStore(): void {
  navigationStore.visibility = { ...allEnabled };
}

export function refreshNavigationVisibility(): void {
  const mode = workspaceStore.mode;
  const isAdmin = authStore.capabilities.admin_console;
  const canWriteResource = authStore.capabilities.resource_write;

  const visibility: MenuVisibility = { ...allEnabled };

  if (mode === "local") {
    visibility.remote_account = "hidden";
    visibility.remote_members_roles = "hidden";
    visibility.remote_permissions_audit = "hidden";
  } else {
    visibility.remote_account = "enabled";
    visibility.remote_members_roles = isAdmin ? "enabled" : "hidden";
    visibility.remote_permissions_audit = isAdmin ? "enabled" : "hidden";
  }

  const sharedVisibility: PermissionVisibility = canWriteResource ? "enabled" : "readonly";
  visibility.workspace_project_config = sharedVisibility;
  visibility.workspace_agent = sharedVisibility;
  visibility.workspace_model = sharedVisibility;
  visibility.workspace_rules = sharedVisibility;
  visibility.workspace_skills = sharedVisibility;
  visibility.workspace_mcp = sharedVisibility;

  navigationStore.visibility = visibility;
}

export function getMenuVisibility(key: MenuKey): PermissionVisibility {
  return navigationStore.visibility[key];
}
