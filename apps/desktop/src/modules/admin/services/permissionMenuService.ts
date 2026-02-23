import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import type { AdminMenu, AdminPermission, PermissionVisibility, Role, RoleMenuVisibility } from "@/shared/types/api";

const mockPermissionsByWorkspace = new Map<string, AdminPermission[]>();
const mockMenusByWorkspace = new Map<string, AdminMenu[]>();
const mockMenuVisibilityByWorkspaceRole = new Map<string, Record<string, PermissionVisibility>>();

export async function listAdminPermissions(workspaceId: string): Promise<AdminPermission[]> {
  return withApiFallback(
    "admin.listPermissions",
    () => getControlClient().get<AdminPermission[]>(`/v1/admin/permissions?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => cloneAdminPermissions(ensureMockPermissions(workspaceId))
  );
}

export async function upsertAdminPermission(workspaceId: string, input: AdminPermission): Promise<AdminPermission> {
  return withApiFallback(
    "admin.upsertPermission",
    () =>
      getControlClient().post<AdminPermission>(`/v1/admin/permissions?workspace_id=${encodeURIComponent(workspaceId)}`, input),
    () => {
      const items = ensureMockPermissions(workspaceId);
      const index = items.findIndex((item) => item.key === input.key);
      const next = { ...input };
      if (index >= 0) {
        items[index] = next;
        return items[index];
      }
      items.push(next);
      return next;
    }
  );
}

export async function patchAdminPermission(
  workspaceId: string,
  permissionKey: string,
  patch: { label?: string; enabled?: boolean }
): Promise<AdminPermission> {
  return withApiFallback(
    "admin.patchPermission",
    () =>
      getControlClient().request<AdminPermission>(
        `/v1/admin/permissions/${encodeURIComponent(permissionKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "PATCH", body: patch }
      ),
    () => {
      const items = ensureMockPermissions(workspaceId);
      const target = items.find((item) => item.key === permissionKey);
      if (!target) {
        throw new Error("Permission not found");
      }
      if (patch.label !== undefined) {
        target.label = patch.label;
      }
      if (patch.enabled !== undefined) {
        target.enabled = patch.enabled;
      }
      return { ...target };
    }
  );
}

export async function removeAdminPermission(workspaceId: string, permissionKey: string): Promise<void> {
  return withApiFallback(
    "admin.removePermission",
    async () => {
      await getControlClient().request<void>(
        `/v1/admin/permissions/${encodeURIComponent(permissionKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "DELETE" }
      );
    },
    () => {
      const items = ensureMockPermissions(workspaceId).filter((item) => item.key !== permissionKey);
      mockPermissionsByWorkspace.set(workspaceId, items);
    }
  );
}

export async function listAdminMenus(workspaceId: string): Promise<AdminMenu[]> {
  return withApiFallback(
    "admin.listMenus",
    () => getControlClient().get<AdminMenu[]>(`/v1/admin/menus?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => cloneAdminMenus(ensureMockMenus(workspaceId))
  );
}

export async function upsertAdminMenu(workspaceId: string, input: AdminMenu): Promise<AdminMenu> {
  return withApiFallback(
    "admin.upsertMenu",
    () => getControlClient().post<AdminMenu>(`/v1/admin/menus?workspace_id=${encodeURIComponent(workspaceId)}`, input),
    () => {
      const items = ensureMockMenus(workspaceId);
      const index = items.findIndex((item) => item.key === input.key);
      const next = { ...input };
      if (index >= 0) {
        items[index] = next;
        return items[index];
      }
      items.push(next);
      return next;
    }
  );
}

export async function patchAdminMenu(
  workspaceId: string,
  menuKey: string,
  patch: { label?: string; enabled?: boolean }
): Promise<AdminMenu> {
  return withApiFallback(
    "admin.patchMenu",
    () =>
      getControlClient().request<AdminMenu>(
        `/v1/admin/menus/${encodeURIComponent(menuKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "PATCH", body: patch }
      ),
    () => {
      const items = ensureMockMenus(workspaceId);
      const target = items.find((item) => item.key === menuKey);
      if (!target) {
        throw new Error("Menu not found");
      }
      if (patch.label !== undefined) {
        target.label = patch.label;
      }
      if (patch.enabled !== undefined) {
        target.enabled = patch.enabled;
      }
      return { ...target };
    }
  );
}

export async function removeAdminMenu(workspaceId: string, menuKey: string): Promise<void> {
  return withApiFallback(
    "admin.removeMenu",
    async () => {
      await getControlClient().request<void>(
        `/v1/admin/menus/${encodeURIComponent(menuKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "DELETE" }
      );
    },
    () => {
      const items = ensureMockMenus(workspaceId).filter((item) => item.key !== menuKey);
      mockMenusByWorkspace.set(workspaceId, items);
    }
  );
}

export async function getRoleMenuVisibility(workspaceId: string, roleKey: Role): Promise<RoleMenuVisibility> {
  return withApiFallback(
    "admin.getRoleMenuVisibility",
    () =>
      getControlClient().get<RoleMenuVisibility>(
        `/v1/admin/menu-visibility/${encodeURIComponent(roleKey)}?workspace_id=${encodeURIComponent(workspaceId)}`
      ),
    () => ({
      role_key: roleKey,
      items: { ...ensureMockRoleMenuVisibility(workspaceId, roleKey) }
    })
  );
}

export async function setRoleMenuVisibility(
  workspaceId: string,
  roleKey: Role,
  items: Record<string, PermissionVisibility>
): Promise<RoleMenuVisibility> {
  const payload: RoleMenuVisibility = { role_key: roleKey, items };
  return withApiFallback(
    "admin.setRoleMenuVisibility",
    () =>
      getControlClient().request<RoleMenuVisibility>(
        `/v1/admin/menu-visibility/${encodeURIComponent(roleKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
        { method: "PUT", body: payload }
      ),
    () => {
      mockMenuVisibilityByWorkspaceRole.set(`${workspaceId}:${roleKey}`, { ...items });
      return { role_key: roleKey, items: { ...items } };
    }
  );
}

function ensureMockPermissions(workspaceId: string): AdminPermission[] {
  const cached = mockPermissionsByWorkspace.get(workspaceId);
  if (cached) {
    return cached;
  }
  const created: AdminPermission[] = [
    { key: "project.read", label: "读取项目", enabled: true },
    { key: "project.write", label: "写入项目", enabled: true },
    { key: "conversation.read", label: "读取会话", enabled: true },
    { key: "conversation.write", label: "写入会话", enabled: true },
    { key: "execution.control", label: "执行控制", enabled: true },
    { key: "resource.read", label: "读取资源", enabled: true },
    { key: "resource.write", label: "写入资源", enabled: true },
    { key: "share.request", label: "发起共享", enabled: true },
    { key: "share.approve", label: "审批共享", enabled: true },
    { key: "share.reject", label: "拒绝共享", enabled: true },
    { key: "share.revoke", label: "撤销共享", enabled: true },
    { key: "model_catalog.sync", label: "同步模型目录", enabled: true },
    { key: "admin.users.manage", label: "成员管理", enabled: true },
    { key: "admin.roles.manage", label: "角色管理", enabled: true },
    { key: "admin.permissions.manage", label: "权限管理", enabled: true },
    { key: "admin.menus.manage", label: "菜单管理", enabled: true },
    { key: "admin.policies.manage", label: "策略管理", enabled: true },
    { key: "admin.audit.read", label: "审计读取", enabled: true }
  ];
  mockPermissionsByWorkspace.set(workspaceId, created);
  return created;
}

function ensureMockMenus(workspaceId: string): AdminMenu[] {
  const cached = mockMenusByWorkspace.get(workspaceId);
  if (cached) {
    return cached;
  }
  const created: AdminMenu[] = [
    { key: "main", label: "主界面", enabled: true },
    { key: "remote_account", label: "账号信息", enabled: true },
    { key: "remote_members_roles", label: "成员与角色", enabled: true },
    { key: "remote_permissions_audit", label: "权限与审计", enabled: true },
    { key: "workspace_project_config", label: "项目配置", enabled: true },
    { key: "workspace_agent", label: "Agent配置", enabled: true },
    { key: "workspace_model", label: "模型配置", enabled: true },
    { key: "workspace_rules", label: "规则配置", enabled: true },
    { key: "workspace_skills", label: "技能配置", enabled: true },
    { key: "workspace_mcp", label: "MCP配置", enabled: true },
    { key: "settings_theme", label: "主题", enabled: true },
    { key: "settings_i18n", label: "国际化", enabled: true },
    { key: "settings_general", label: "通用设置", enabled: true }
  ];
  mockMenusByWorkspace.set(workspaceId, created);
  return created;
}

function ensureMockRoleMenuVisibility(workspaceId: string, roleKey: Role): Record<string, PermissionVisibility> {
  const cacheKey = `${workspaceId}:${roleKey}`;
  const cached = mockMenuVisibilityByWorkspaceRole.get(cacheKey);
  if (cached) {
    return cached;
  }
  const allMenus = ensureMockMenus(workspaceId);
  const visibility: Record<string, PermissionVisibility> = {};
  for (const menu of allMenus) {
    visibility[menu.key] = "enabled";
  }
  if (roleKey !== "admin") {
    visibility.remote_members_roles = "hidden";
    visibility.remote_permissions_audit = roleKey === "approver" ? "enabled" : "hidden";
  }
  mockMenuVisibilityByWorkspaceRole.set(cacheKey, visibility);
  return visibility;
}

function cloneAdminPermissions(items: AdminPermission[]): AdminPermission[] {
  return items.map((item) => ({ ...item }));
}

function cloneAdminMenus(items: AdminMenu[]): AdminMenu[] {
  return items.map((item) => ({ ...item }));
}
