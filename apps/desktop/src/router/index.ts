import { createRouter, createWebHistory, type RouteRecordRaw, type RouterHistory } from "vue-router";

import { initializeWorkspaceContext, switchWorkspaceContext } from "@/modules/workspace/store";
import { isRuntimeCapabilitySupported, type RuntimeCapabilities } from "@/shared/runtime";
import { canAccessAdmin } from "@/shared/stores/authStore";
import { getMenuVisibility, refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { MenuKey } from "@/shared/types/api";

type RouteMeta = {
  menuKey?: MenuKey;
  requiresRemote?: boolean;
  requiresAdmin?: boolean;
  requiresCapability?: keyof RuntimeCapabilities;
};

export const routes: RouteRecordRaw[] = [
  { path: "/", redirect: "/main" },
  {
    path: "/main",
    name: "main",
    component: () => import("@/modules/conversation/views/MainScreenView.vue"),
    meta: { menuKey: "main" } as RouteMeta
  },
  {
    path: "/remote/account",
    name: "remote-account",
    component: () => import("@/modules/admin/views/RemoteAccountView.vue"),
    meta: { menuKey: "remote_account", requiresRemote: true } as RouteMeta
  },
  {
    path: "/remote/members-roles",
    name: "remote-members-roles",
    component: () => import("@/modules/admin/views/RemoteMembersRolesView.vue"),
    meta: { menuKey: "remote_members_roles", requiresRemote: true, requiresAdmin: true } as RouteMeta
  },
  {
    path: "/remote/permissions-audit",
    name: "remote-permissions-audit",
    component: () => import("@/modules/admin/views/RemotePermissionsAuditView.vue"),
    meta: { menuKey: "remote_permissions_audit", requiresRemote: true, requiresAdmin: true } as RouteMeta
  },
  {
    path: "/workspace/agent",
    name: "workspace-agent",
    component: () => import("@/modules/resource/views/WorkspaceAgentView.vue"),
    meta: { menuKey: "workspace_agent" } as RouteMeta
  },
  {
    path: "/workspace/project-config",
    name: "workspace-project-config",
    component: () => import("@/modules/resource/views/WorkspaceProjectConfigView.vue"),
    meta: { menuKey: "workspace_project_config" } as RouteMeta
  },
  {
    path: "/workspace/model",
    name: "workspace-model",
    component: () => import("@/modules/resource/views/WorkspaceModelView.vue"),
    meta: { menuKey: "workspace_model" } as RouteMeta
  },
  {
    path: "/workspace/rules",
    name: "workspace-rules",
    component: () => import("@/modules/resource/views/WorkspaceRulesView.vue"),
    meta: { menuKey: "workspace_rules" } as RouteMeta
  },
  {
    path: "/workspace/skills",
    name: "workspace-skills",
    component: () => import("@/modules/resource/views/WorkspaceSkillsView.vue"),
    meta: { menuKey: "workspace_skills" } as RouteMeta
  },
  {
    path: "/workspace/mcp",
    name: "workspace-mcp",
    component: () => import("@/modules/resource/views/WorkspaceMcpView.vue"),
    meta: { menuKey: "workspace_mcp" } as RouteMeta
  },
  {
    path: "/settings/theme",
    name: "settings-theme",
    component: () => import("@/modules/workspace/views/SettingsThemeView.vue"),
    meta: { menuKey: "settings_theme", requiresCapability: "supportsLocalWorkspace" } as RouteMeta
  },
  {
    path: "/settings/i18n",
    name: "settings-i18n",
    component: () => import("@/modules/workspace/views/SettingsI18nView.vue"),
    meta: { menuKey: "settings_i18n", requiresCapability: "supportsLocalWorkspace" } as RouteMeta
  },
  {
    path: "/settings/general",
    name: "settings-general",
    component: () => import("@/modules/workspace/views/SettingsGeneralView.vue"),
    meta: { menuKey: "settings_general", requiresCapability: "supportsLocalWorkspace" } as RouteMeta
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
    if (meta.requiresCapability && !isRuntimeCapabilitySupported(meta.requiresCapability)) {
      return { path: "/main", query: { reason: "capability_required" } };
    }

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

    let visibility: "hidden" | "disabled" | "readonly" | "enabled" = "enabled";
    if (menuKey) {
      visibility = getMenuVisibility(menuKey);
      if (visibility === "hidden") {
        return { path: "/main", query: { reason: "menu_hidden" } };
      }
      if (visibility === "disabled") {
        return { path: to.path.startsWith("/remote/") ? "/remote/account" : "/main", query: { reason: "menu_disabled" } };
      }
    }

    if (meta.requiresAdmin && !canAccessAdmin() && visibility === "enabled") {
      return { path: "/remote/account", query: { reason: "admin_forbidden" } };
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
