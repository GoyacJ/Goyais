import { createRouter, createWebHistory, type RouteRecordRaw, type RouterHistory } from "vue-router";

import RemoteAccountView from "@/modules/admin/views/RemoteAccountView.vue";
import RemoteMembersRolesView from "@/modules/admin/views/RemoteMembersRolesView.vue";
import RemotePermissionsAuditView from "@/modules/admin/views/RemotePermissionsAuditView.vue";
import MainScreenView from "@/modules/conversation/views/MainScreenView.vue";
import WorkspaceAgentView from "@/modules/resource/views/WorkspaceAgentView.vue";
import WorkspaceMcpView from "@/modules/resource/views/WorkspaceMcpView.vue";
import WorkspaceModelView from "@/modules/resource/views/WorkspaceModelView.vue";
import WorkspaceProjectConfigView from "@/modules/resource/views/WorkspaceProjectConfigView.vue";
import WorkspaceRulesView from "@/modules/resource/views/WorkspaceRulesView.vue";
import WorkspaceSkillsView from "@/modules/resource/views/WorkspaceSkillsView.vue";
import { initializeWorkspaceContext, switchWorkspaceContext } from "@/modules/workspace/store";
import SettingsGeneralView from "@/modules/workspace/views/SettingsGeneralView.vue";
import SettingsI18nView from "@/modules/workspace/views/SettingsI18nView.vue";
import SettingsThemeView from "@/modules/workspace/views/SettingsThemeView.vue";
import SettingsUpdatesDiagnosticsView from "@/modules/workspace/views/SettingsUpdatesDiagnosticsView.vue";
import { canAccessAdmin } from "@/shared/stores/authStore";
import { getMenuVisibility, refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { MenuKey } from "@/shared/types/api";

type RouteMeta = {
  menuKey?: MenuKey;
  requiresRemote?: boolean;
  requiresAdmin?: boolean;
};

export const routes: RouteRecordRaw[] = [
  { path: "/", redirect: "/main" },
  { path: "/main", name: "main", component: MainScreenView, meta: { menuKey: "main" } as RouteMeta },
  {
    path: "/remote/account",
    name: "remote-account",
    component: RemoteAccountView,
    meta: { menuKey: "remote_account", requiresRemote: true } as RouteMeta
  },
  {
    path: "/remote/members-roles",
    name: "remote-members-roles",
    component: RemoteMembersRolesView,
    meta: { menuKey: "remote_members_roles", requiresRemote: true, requiresAdmin: true } as RouteMeta
  },
  {
    path: "/remote/permissions-audit",
    name: "remote-permissions-audit",
    component: RemotePermissionsAuditView,
    meta: { menuKey: "remote_permissions_audit", requiresRemote: true, requiresAdmin: true } as RouteMeta
  },
  {
    path: "/workspace/agent",
    name: "workspace-agent",
    component: WorkspaceAgentView,
    meta: { menuKey: "workspace_agent" } as RouteMeta
  },
  {
    path: "/workspace/project-config",
    name: "workspace-project-config",
    component: WorkspaceProjectConfigView,
    meta: { menuKey: "workspace_project_config" } as RouteMeta
  },
  {
    path: "/workspace/model",
    name: "workspace-model",
    component: WorkspaceModelView,
    meta: { menuKey: "workspace_model" } as RouteMeta
  },
  {
    path: "/workspace/rules",
    name: "workspace-rules",
    component: WorkspaceRulesView,
    meta: { menuKey: "workspace_rules" } as RouteMeta
  },
  {
    path: "/workspace/skills",
    name: "workspace-skills",
    component: WorkspaceSkillsView,
    meta: { menuKey: "workspace_skills" } as RouteMeta
  },
  {
    path: "/workspace/mcp",
    name: "workspace-mcp",
    component: WorkspaceMcpView,
    meta: { menuKey: "workspace_mcp" } as RouteMeta
  },
  {
    path: "/settings/theme",
    name: "settings-theme",
    component: SettingsThemeView,
    meta: { menuKey: "settings_theme" } as RouteMeta
  },
  {
    path: "/settings/i18n",
    name: "settings-i18n",
    component: SettingsI18nView,
    meta: { menuKey: "settings_i18n" } as RouteMeta
  },
  {
    path: "/settings/updates-diagnostics",
    name: "settings-updates-diagnostics",
    component: SettingsUpdatesDiagnosticsView,
    meta: { menuKey: "settings_updates_diagnostics" } as RouteMeta
  },
  {
    path: "/settings/general",
    name: "settings-general",
    component: SettingsGeneralView,
    meta: { menuKey: "settings_general" } as RouteMeta
  }
];

let initialized = false;

export function createAppRouter(history: RouterHistory = createWebHistory()) {
  const appRouter = createRouter({
    history,
    routes
  });

  appRouter.beforeEach(async (to) => {
    if (!initialized) {
      await initializeWorkspaceContext();
      initialized = true;
    }

    refreshNavigationVisibility();
    const meta = to.meta as RouteMeta;
    const menuKey = meta.menuKey;

    if (to.path.startsWith("/settings/") && workspaceStore.mode !== "local") {
      const switched = await trySwitchToLocalWorkspace();
      if (!switched) {
        return { path: "/main", query: { reason: "local_required" } };
      }
      refreshNavigationVisibility();
    }

    if (meta.requiresRemote && workspaceStore.mode !== "remote") {
      const switched = await trySwitchToRemoteWorkspace();
      if (!switched) {
        return { path: "/main", query: { reason: "remote_required" } };
      }
    }

    if (meta.requiresAdmin && !canAccessAdmin()) {
      return { path: "/remote/account", query: { reason: "admin_forbidden" } };
    }

    if (menuKey) {
      const visibility = getMenuVisibility(menuKey);
      if (visibility === "hidden") {
        return { path: "/main", query: { reason: "menu_hidden" } };
      }
    }

    return true;
  });

  return appRouter;
}

export const router = createAppRouter();

export function resetRouterInitForTests(value = false): void {
  initialized = value;
}

async function trySwitchToRemoteWorkspace(): Promise<boolean> {
  const remoteWorkspace = workspaceStore.workspaces.find((workspace) => workspace.mode === "remote");
  if (!remoteWorkspace) {
    return false;
  }

  await switchWorkspaceContext(remoteWorkspace.id);
  return workspaceStore.mode === "remote";
}

async function trySwitchToLocalWorkspace(): Promise<boolean> {
  const localWorkspace = workspaceStore.workspaces.find(
    (workspace) => workspace.mode === "local" || workspace.is_default_local
  );
  if (!localWorkspace) {
    return false;
  }

  await switchWorkspaceContext(localWorkspace.id);
  return workspaceStore.mode === "local";
}
