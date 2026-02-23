import { reactive } from "vue";

import {
  listAdminRoles,
  listAdminUsers,
  listAuditEvents,
  removeAdminRole,
  removeAdminUser,
  setAdminRoleEnabled,
  setAdminUserEnabled,
  setAdminUserRole,
  upsertAdminRole,
  upsertAdminUser
} from "@/modules/admin/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { createMockId } from "@/shared/services/mockData";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { AdminAuditEvent, AdminRole, AdminUser, PermissionVisibility, Role } from "@/shared/types/api";

export type MenuPermissionNode = {
  id: string;
  label: string;
  visibility: PermissionVisibility;
  enabled: boolean;
};

export type ActionPermissionItem = {
  id: string;
  label: string;
  enabled: boolean;
};

type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

type AdminState = {
  users: AdminUser[];
  usersPage: CursorPageState;
  roles: AdminRole[];
  audits: AdminAuditEvent[];
  auditsPage: CursorPageState;
  menuNodes: MenuPermissionNode[];
  permissionItems: ActionPermissionItem[];
  loading: boolean;
  error: string;
};

const initialState: AdminState = {
  users: [],
  usersPage: createInitialPageState(),
  roles: [],
  audits: [],
  auditsPage: createInitialPageState(),
  menuNodes: createDefaultMenuNodes(),
  permissionItems: createDefaultPermissionItems(),
  loading: false,
  error: ""
};

export const adminStore = reactive<AdminState>({ ...initialState });

export function resetAdminStore(): void {
  adminStore.users = [];
  adminStore.usersPage = createInitialPageState();
  adminStore.roles = [];
  adminStore.audits = [];
  adminStore.auditsPage = createInitialPageState();
  adminStore.menuNodes = createDefaultMenuNodes();
  adminStore.permissionItems = createDefaultPermissionItems();
  adminStore.loading = false;
  adminStore.error = "";
}

export async function refreshAdminData(): Promise<void> {
  await refreshAdminUsersPage({ cursor: null, backStack: [] });
  await refreshAdminAuditPage({ cursor: null, backStack: [] });
}

export async function loadNextAdminUsersPage(): Promise<void> {
  const nextCursor = adminStore.usersPage.nextCursor;
  if (!nextCursor || adminStore.usersPage.loading) {
    return;
  }
  await refreshAdminUsersPage({
    cursor: nextCursor,
    backStack: [...adminStore.usersPage.backStack, adminStore.usersPage.currentCursor]
  });
}

export async function loadPreviousAdminUsersPage(): Promise<void> {
  if (adminStore.usersPage.backStack.length === 0 || adminStore.usersPage.loading) {
    return;
  }
  const backStack = [...adminStore.usersPage.backStack];
  const previousCursor = backStack.pop() ?? null;
  await refreshAdminUsersPage({ cursor: previousCursor, backStack });
}

export async function loadNextAdminAuditsPage(): Promise<void> {
  const nextCursor = adminStore.auditsPage.nextCursor;
  if (!nextCursor || adminStore.auditsPage.loading) {
    return;
  }
  await refreshAdminAuditPage({
    cursor: nextCursor,
    backStack: [...adminStore.auditsPage.backStack, adminStore.auditsPage.currentCursor]
  });
}

export async function loadPreviousAdminAuditsPage(): Promise<void> {
  if (adminStore.auditsPage.backStack.length === 0 || adminStore.auditsPage.loading) {
    return;
  }
  const backStack = [...adminStore.auditsPage.backStack];
  const previousCursor = backStack.pop() ?? null;
  await refreshAdminAuditPage({ cursor: previousCursor, backStack });
}

async function refreshAdminUsersPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || workspace.mode !== "remote") {
    return;
  }

  adminStore.loading = true;
  adminStore.usersPage.loading = true;
  adminStore.error = "";

  try {
    const [users, roles] = await Promise.all([
      listAdminUsers(workspace.id, {
        cursor: input.cursor ?? undefined,
        limit: adminStore.usersPage.limit
      }),
      listAdminRoles()
    ]);

    adminStore.users = users.items;
    adminStore.usersPage.currentCursor = input.cursor;
    adminStore.usersPage.backStack = input.backStack;
    adminStore.usersPage.nextCursor = users.next_cursor;
    adminStore.roles = roles;
  } catch (error) {
    adminStore.error = toDisplayError(error);
  } finally {
    adminStore.usersPage.loading = false;
    adminStore.loading = false;
  }
}

async function refreshAdminAuditPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || workspace.mode !== "remote") {
    return;
  }

  adminStore.auditsPage.loading = true;
  try {
    const audits = await listAuditEvents(workspace.id, {
      cursor: input.cursor ?? undefined,
      limit: adminStore.auditsPage.limit
    });
    adminStore.audits = audits.items;
    adminStore.auditsPage.currentCursor = input.cursor;
    adminStore.auditsPage.backStack = input.backStack;
    adminStore.auditsPage.nextCursor = audits.next_cursor;
  } catch (error) {
    adminStore.error = toDisplayError(error);
  } finally {
    adminStore.auditsPage.loading = false;
    adminStore.loading = false;
  }
}

export async function createOrUpdateAdminUser(input: {
  username: string;
  display_name: string;
  role: Role;
}): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || workspace.mode !== "remote") {
    return;
  }

  try {
    const created = await upsertAdminUser(workspace.id, input);
    const index = adminStore.users.findIndex((user) => user.id === created.id);
    if (index >= 0) {
      adminStore.users[index] = created;
    } else {
      adminStore.users.push(created);
    }
    pushAudit("user.upsert", created.username, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteAdminUser(userId: string): Promise<void> {
  try {
    const target = adminStore.users.find((user) => user.id === userId);
    await removeAdminUser(userId);
    adminStore.users = adminStore.users.filter((user) => user.id !== userId);
    pushAudit("user.delete", target?.username ?? userId, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function toggleAdminUser(userId: string): Promise<void> {
  const target = adminStore.users.find((user) => user.id === userId);
  if (!target) {
    return;
  }

  try {
    const updated = await setAdminUserEnabled(userId, !target.enabled);
    applyUserUpdate(updated);
    pushAudit("user.toggle", updated.username, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function assignRoleToUser(userId: string, role: Role): Promise<void> {
  try {
    const updated = await setAdminUserRole(userId, role);
    applyUserUpdate(updated);
    pushAudit("user.assign_role", `${updated.username}:${role}`, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function createOrUpdateRole(input: AdminRole): Promise<void> {
  try {
    const updated = await upsertAdminRole(input);
    const index = adminStore.roles.findIndex((role) => role.key === updated.key);
    if (index >= 0) {
      adminStore.roles[index] = updated;
    } else {
      adminStore.roles.push(updated);
    }
    pushAudit("role.upsert", updated.key, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteRole(roleKey: Role): Promise<void> {
  try {
    await removeAdminRole(roleKey);
    adminStore.roles = adminStore.roles.filter((role) => role.key !== roleKey);
    pushAudit("role.delete", roleKey, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function toggleRole(roleKey: Role): Promise<void> {
  const target = adminStore.roles.find((role) => role.key === roleKey);
  if (!target) {
    return;
  }

  try {
    const updated = await setAdminRoleEnabled(roleKey, !target.enabled);
    const index = adminStore.roles.findIndex((role) => role.key === roleKey);
    if (index >= 0) {
      adminStore.roles[index] = updated;
    }
    pushAudit("role.toggle", roleKey, "success");
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export function assignPermissionToRole(roleKey: Role, permission: string): void {
  const target = adminStore.roles.find((role) => role.key === roleKey);
  if (!target || target.permissions.includes(permission)) {
    return;
  }
  target.permissions = [...target.permissions, permission];
  pushAudit("role.assign_permission", `${roleKey}:${permission}`, "success");
}

export function removePermissionFromRole(roleKey: Role, permission: string): void {
  const target = adminStore.roles.find((role) => role.key === roleKey);
  if (!target) {
    return;
  }
  target.permissions = target.permissions.filter((item) => item !== permission);
  pushAudit("role.remove_permission", `${roleKey}:${permission}`, "success");
}

export function toggleMenuNode(menuId: string): void {
  const target = adminStore.menuNodes.find((item) => item.id === menuId);
  if (!target) {
    return;
  }
  target.enabled = !target.enabled;
  target.visibility = target.enabled ? "enabled" : "disabled";
  pushAudit("menu.toggle", menuId, "success");
}

export function deleteMenuNode(menuId: string): void {
  adminStore.menuNodes = adminStore.menuNodes.filter((item) => item.id !== menuId);
  pushAudit("menu.delete", menuId, "success");
}

export function togglePermissionItem(permissionId: string): void {
  const target = adminStore.permissionItems.find((item) => item.id === permissionId);
  if (!target) {
    return;
  }
  target.enabled = !target.enabled;
  pushAudit("permission.toggle", permissionId, "success");
}

export function deletePermissionItem(permissionId: string): void {
  adminStore.permissionItems = adminStore.permissionItems.filter((item) => item.id !== permissionId);
  pushAudit("permission.delete", permissionId, "success");
}

function applyUserUpdate(user: AdminUser): void {
  const index = adminStore.users.findIndex((item) => item.id === user.id);
  if (index >= 0) {
    adminStore.users[index] = user;
  }
}

function pushAudit(action: string, resource: string, result: AdminAuditEvent["result"]): void {
  adminStore.audits = [
    {
      id: createMockId("audit"),
      actor: "ui-admin",
      action,
      resource,
      result,
      trace_id: createMockId("tr"),
      timestamp: new Date().toISOString()
    },
    ...adminStore.audits
  ];
}

function createDefaultMenuNodes(): MenuPermissionNode[] {
  return [
    { id: "main", label: "主屏幕", visibility: "enabled", enabled: true },
    { id: "remote_account", label: "账号信息", visibility: "enabled", enabled: true },
    { id: "remote_members_roles", label: "成员与角色", visibility: "enabled", enabled: true },
    { id: "remote_permissions_audit", label: "权限与审计", visibility: "enabled", enabled: true },
    { id: "workspace_project_config", label: "项目配置", visibility: "enabled", enabled: true }
  ];
}

function createDefaultPermissionItems(): ActionPermissionItem[] {
  return [
    { id: "execution.run", label: "执行任务", enabled: true },
    { id: "execution.stop", label: "停止任务", enabled: true },
    { id: "resource.share", label: "共享资源", enabled: true },
    { id: "resource.approve", label: "审批共享", enabled: true },
    { id: "audit.read", label: "查看审计日志", enabled: true }
  ];
}

function createInitialPageState(limit = 20): CursorPageState {
  return {
    limit,
    currentCursor: null,
    backStack: [],
    nextCursor: null,
    loading: false
  };
}
