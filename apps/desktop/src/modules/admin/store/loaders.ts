import { listABACPolicies, listAdminMenus, listAdminPermissions, listAdminRoles, listAdminUsers, listAuditEvents } from "@/modules/admin/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import type { PermissionVisibility, Role } from "@/shared/types/api";

import { adminStore, getRemoteWorkspaceId } from "./state";
import { getRoleMenuVisibility, setRoleMenuVisibility } from "@/modules/admin/services";

export async function refreshAdminData(): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }

  adminStore.loading = true;
  adminStore.error = "";
  try {
    await Promise.all([
      refreshAdminUsersPage({ cursor: null, backStack: [] }),
      refreshAdminAuditPage({ cursor: null, backStack: [] }),
      refreshAdminMetadata(workspaceId)
    ]);
    await refreshRoleMenuVisibility(adminStore.activeRoleKey);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  } finally {
    adminStore.loading = false;
  }
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

export async function refreshRoleMenuVisibility(roleKey: Role): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  adminStore.activeRoleKey = roleKey;
  try {
    const payload = await getRoleMenuVisibility(workspaceId, roleKey);
    adminStore.roleMenuVisibility = payload.items;
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function saveRoleMenuVisibilityForActiveRole(): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    const payload = await setRoleMenuVisibility(workspaceId, adminStore.activeRoleKey, adminStore.roleMenuVisibility);
    adminStore.roleMenuVisibility = payload.items;
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export function setMenuVisibilityItem(menuKey: string, visibility: PermissionVisibility): void {
  adminStore.roleMenuVisibility = {
    ...adminStore.roleMenuVisibility,
    [menuKey]: visibility
  };
}

async function refreshAdminUsersPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }

  adminStore.usersPage.loading = true;
  try {
    const users = await listAdminUsers(workspaceId, {
      cursor: input.cursor ?? undefined,
      limit: adminStore.usersPage.limit
    });
    adminStore.users = users.items;
    adminStore.usersPage.currentCursor = input.cursor;
    adminStore.usersPage.backStack = input.backStack;
    adminStore.usersPage.nextCursor = users.next_cursor;
  } finally {
    adminStore.usersPage.loading = false;
  }
}

async function refreshAdminAuditPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }

  adminStore.auditsPage.loading = true;
  try {
    const audits = await listAuditEvents(workspaceId, {
      cursor: input.cursor ?? undefined,
      limit: adminStore.auditsPage.limit
    });
    adminStore.audits = audits.items;
    adminStore.auditsPage.currentCursor = input.cursor;
    adminStore.auditsPage.backStack = input.backStack;
    adminStore.auditsPage.nextCursor = audits.next_cursor;
  } finally {
    adminStore.auditsPage.loading = false;
  }
}

async function refreshAdminMetadata(workspaceId: string): Promise<void> {
  const [roles, permissions, menus, policies] = await Promise.all([
    listAdminRoles(workspaceId),
    listAdminPermissions(workspaceId),
    listAdminMenus(workspaceId),
    listABACPolicies(workspaceId)
  ]);
  adminStore.roles = roles;
  adminStore.permissionItems = permissions;
  adminStore.menuNodes = menus;
  adminStore.abacPolicies = policies;
}
