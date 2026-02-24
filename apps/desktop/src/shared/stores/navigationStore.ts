import { reactive } from "vue";

import { authStore } from "@/shared/stores/authStore";
import { getCurrentPermissionSnapshot } from "@/shared/stores/permissionStore";
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
  const snapshot = getCurrentPermissionSnapshot();

  const visibility: MenuVisibility = { ...allEnabled };

  if (mode === "local") {
    hideRemoteAdminMenus(visibility);
  } else {
    if (snapshot) {
      applyVisibilityFromSnapshot(visibility, snapshot.menu_visibility);
    } else {
      applyDefaultRemoteVisibility(visibility, isAdmin, canWriteResource);
    }
  }

  enforceSettingsMenusAlwaysEnabled(visibility);

  navigationStore.visibility = visibility;
}

export function getMenuVisibility(key: MenuKey): PermissionVisibility {
  return navigationStore.visibility[key];
}

function isPermissionVisibility(value: unknown): value is PermissionVisibility {
  return value === "hidden" || value === "disabled" || value === "readonly" || value === "enabled";
}

function hideRemoteAdminMenus(visibility: MenuVisibility): void {
  visibility.remote_account = "hidden";
  visibility.remote_members_roles = "hidden";
  visibility.remote_permissions_audit = "hidden";
}

function applyVisibilityFromSnapshot(
  visibility: MenuVisibility,
  snapshot: Record<string, PermissionVisibility>
): void {
  for (const key of Object.keys(visibility) as MenuKey[]) {
    if (settingsMenuKeys.includes(key)) {
      continue;
    }
    const value = snapshot[key];
    if (isPermissionVisibility(value)) {
      visibility[key] = value;
    }
  }
}

function applyDefaultRemoteVisibility(visibility: MenuVisibility, isAdmin: boolean, canWriteResource: boolean): void {
  visibility.remote_account = "enabled";
  visibility.remote_members_roles = isAdmin ? "enabled" : "hidden";
  visibility.remote_permissions_audit = isAdmin ? "enabled" : "hidden";
  const sharedVisibility: PermissionVisibility = canWriteResource ? "enabled" : "readonly";
  visibility.workspace_project_config = sharedVisibility;
  visibility.workspace_agent = sharedVisibility;
  visibility.workspace_model = sharedVisibility;
  visibility.workspace_rules = sharedVisibility;
  visibility.workspace_skills = sharedVisibility;
  visibility.workspace_mcp = sharedVisibility;
}

function enforceSettingsMenusAlwaysEnabled(visibility: MenuVisibility): void {
  for (const key of settingsMenuKeys) {
    visibility[key] = "enabled";
  }
}

const settingsMenuKeys: MenuKey[] = ["settings_theme", "settings_i18n", "settings_general"];
