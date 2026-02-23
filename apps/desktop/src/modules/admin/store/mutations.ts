import {
  deleteABACPolicy as removeABACPolicy,
  patchABACPolicy,
  patchAdminMenu,
  patchAdminPermission,
  removeAdminMenu,
  removeAdminPermission,
  removeAdminRole,
  removeAdminUser,
  setAdminRoleEnabled,
  setAdminUserEnabled,
  setAdminUserRole,
  upsertABACPolicy,
  upsertAdminMenu,
  upsertAdminPermission,
  upsertAdminRole,
  upsertAdminUser
} from "@/modules/admin/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import type { ABACPolicy, AdminMenu, AdminPermission, AdminRole, Role } from "@/shared/types/api";

import { adminStore, applyUserUpdate, getRemoteWorkspaceId, upsertMenu, upsertPermission, upsertPolicy } from "./state";

export async function createOrUpdateAdminUser(input: {
  username: string;
  display_name: string;
  role: Role;
}): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }

  try {
    const created = await upsertAdminUser(workspaceId, input);
    const index = adminStore.users.findIndex((user) => user.id === created.id);
    if (index >= 0) {
      adminStore.users[index] = created;
    } else {
      adminStore.users.push(created);
    }
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteAdminUser(userId: string): Promise<void> {
  try {
    await removeAdminUser(userId);
    adminStore.users = adminStore.users.filter((user) => user.id !== userId);
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
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function assignRoleToUser(userId: string, role: Role): Promise<void> {
  try {
    const updated = await setAdminUserRole(userId, role);
    applyUserUpdate(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function createOrUpdateRole(input: AdminRole): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    const updated = await upsertAdminRole(workspaceId, input);
    const index = adminStore.roles.findIndex((role) => role.key === updated.key);
    if (index >= 0) {
      adminStore.roles[index] = updated;
    } else {
      adminStore.roles.push(updated);
    }
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteRole(roleKey: Role): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    await removeAdminRole(workspaceId, roleKey);
    adminStore.roles = adminStore.roles.filter((role) => role.key !== roleKey);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function toggleRole(roleKey: Role): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  const target = adminStore.roles.find((role) => role.key === roleKey);
  if (!workspaceId || !target) {
    return;
  }
  try {
    const updated = await setAdminRoleEnabled(workspaceId, roleKey, !target.enabled);
    const index = adminStore.roles.findIndex((role) => role.key === roleKey);
    if (index >= 0) {
      adminStore.roles[index] = updated;
    }
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function assignPermissionToRole(roleKey: Role, permission: string): Promise<void> {
  const target = adminStore.roles.find((role) => role.key === roleKey);
  if (!target || target.permissions.includes(permission)) {
    return;
  }
  await createOrUpdateRole({ ...target, permissions: [...target.permissions, permission] });
}

export async function createOrUpdatePermission(input: AdminPermission): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    const updated = await upsertAdminPermission(workspaceId, input);
    upsertPermission(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function togglePermissionItem(permissionKey: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  const target = adminStore.permissionItems.find((item) => item.key === permissionKey);
  if (!workspaceId || !target) {
    return;
  }
  try {
    const updated = await patchAdminPermission(workspaceId, permissionKey, { enabled: !target.enabled });
    upsertPermission(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deletePermissionItem(permissionKey: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    await removeAdminPermission(workspaceId, permissionKey);
    adminStore.permissionItems = adminStore.permissionItems.filter((item) => item.key !== permissionKey);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function createOrUpdateMenuNode(input: AdminMenu): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    const updated = await upsertAdminMenu(workspaceId, input);
    upsertMenu(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function toggleMenuNode(menuKey: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  const target = adminStore.menuNodes.find((item) => item.key === menuKey);
  if (!workspaceId || !target) {
    return;
  }
  try {
    const updated = await patchAdminMenu(workspaceId, menuKey, { enabled: !target.enabled });
    upsertMenu(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteMenuNode(menuKey: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    await removeAdminMenu(workspaceId, menuKey);
    adminStore.menuNodes = adminStore.menuNodes.filter((item) => item.key !== menuKey);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function createOrUpdateABACPolicy(input: ABACPolicy): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    const updated = await upsertABACPolicy(workspaceId, input);
    upsertPolicy(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function toggleABACPolicy(policyId: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  const target = adminStore.abacPolicies.find((item) => item.id === policyId);
  if (!workspaceId || !target) {
    return;
  }
  try {
    const updated = await patchABACPolicy(workspaceId, policyId, { enabled: !target.enabled });
    upsertPolicy(updated);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}

export async function deleteABACPolicy(policyId: string): Promise<void> {
  const workspaceId = getRemoteWorkspaceId();
  if (!workspaceId) {
    return;
  }
  try {
    await removeABACPolicy(workspaceId, policyId);
    adminStore.abacPolicies = adminStore.abacPolicies.filter((item) => item.id !== policyId);
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}
