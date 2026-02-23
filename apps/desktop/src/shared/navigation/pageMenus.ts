import { computed } from "vue";

import { useI18n } from "@/shared/i18n";
import { getMenuVisibility } from "@/shared/stores/navigationStore";
import type { MenuKey, PermissionVisibility } from "@/shared/types/api";

export type MenuEntry = {
  key: MenuKey;
  label: string;
  path: string;
  visibility: PermissionVisibility;
};

export function useRemoteConfigMenu() {
  const { t } = useI18n();

  return computed<MenuEntry[]>(() => [
    createEntry("remote_account", t("menu.remoteAccount"), "/remote/account"),
    createEntry("remote_members_roles", t("menu.remoteMembersRoles"), "/remote/members-roles"),
    createEntry("remote_permissions_audit", t("menu.remotePermissionsAudit"), "/remote/permissions-audit"),
    createEntry("workspace_agent", t("menu.workspaceAgent"), "/workspace/agent"),
    createEntry("workspace_model", t("menu.workspaceModel"), "/workspace/model"),
    createEntry("workspace_rules", t("menu.workspaceRules"), "/workspace/rules"),
    createEntry("workspace_skills", t("menu.workspaceSkills"), "/workspace/skills"),
    createEntry("workspace_mcp", t("menu.workspaceMcp"), "/workspace/mcp")
  ]);
}

export function useLocalSettingsMenu() {
  const { t } = useI18n();

  return computed<MenuEntry[]>(() => [
    createEntry("workspace_agent", t("menu.workspaceAgent"), "/workspace/agent"),
    createEntry("workspace_model", t("menu.workspaceModel"), "/workspace/model"),
    createEntry("workspace_rules", t("menu.workspaceRules"), "/workspace/rules"),
    createEntry("workspace_skills", t("menu.workspaceSkills"), "/workspace/skills"),
    createEntry("workspace_mcp", t("menu.workspaceMcp"), "/workspace/mcp"),
    createEntry("settings_theme", t("menu.settingsTheme"), "/settings/theme"),
    createEntry("settings_i18n", t("menu.settingsI18n"), "/settings/i18n"),
    createEntry("settings_updates_diagnostics", t("menu.settingsUpdatesDiagnostics"), "/settings/updates-diagnostics"),
    createEntry("settings_general", t("menu.settingsGeneral"), "/settings/general")
  ]);
}

function createEntry(key: MenuKey, label: string, path: string): MenuEntry {
  return {
    key,
    label,
    path,
    visibility: getMenuVisibility(key)
  };
}
