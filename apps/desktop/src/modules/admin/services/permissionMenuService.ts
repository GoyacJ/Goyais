import { getControlClient } from "@/shared/services/clients";
import type { AdminMenu, AdminPermission, PermissionVisibility, Role, RoleMenuVisibility } from "@/shared/types/api";

export async function listAdminPermissions(workspaceId: string): Promise<AdminPermission[]> {
  return getControlClient().get<AdminPermission[]>(`/v1/admin/permissions?workspace_id=${encodeURIComponent(workspaceId)}`);
}

export async function upsertAdminPermission(workspaceId: string, input: AdminPermission): Promise<AdminPermission> {
  return getControlClient().post<AdminPermission>(`/v1/admin/permissions?workspace_id=${encodeURIComponent(workspaceId)}`, input);
}

export async function patchAdminPermission(
  workspaceId: string,
  permissionKey: string,
  patch: { label?: string; enabled?: boolean }
): Promise<AdminPermission> {
  return getControlClient().request<AdminPermission>(
    `/v1/admin/permissions/${encodeURIComponent(permissionKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "PATCH", body: patch }
  );
}

export async function removeAdminPermission(workspaceId: string, permissionKey: string): Promise<void> {
  await getControlClient().request<void>(
    `/v1/admin/permissions/${encodeURIComponent(permissionKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "DELETE" }
  );
}

export async function listAdminMenus(workspaceId: string): Promise<AdminMenu[]> {
  return getControlClient().get<AdminMenu[]>(`/v1/admin/menus?workspace_id=${encodeURIComponent(workspaceId)}`);
}

export async function upsertAdminMenu(workspaceId: string, input: AdminMenu): Promise<AdminMenu> {
  return getControlClient().post<AdminMenu>(`/v1/admin/menus?workspace_id=${encodeURIComponent(workspaceId)}`, input);
}

export async function patchAdminMenu(
  workspaceId: string,
  menuKey: string,
  patch: { label?: string; enabled?: boolean }
): Promise<AdminMenu> {
  return getControlClient().request<AdminMenu>(
    `/v1/admin/menus/${encodeURIComponent(menuKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "PATCH", body: patch }
  );
}

export async function removeAdminMenu(workspaceId: string, menuKey: string): Promise<void> {
  await getControlClient().request<void>(
    `/v1/admin/menus/${encodeURIComponent(menuKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "DELETE" }
  );
}

export async function getRoleMenuVisibility(workspaceId: string, roleKey: Role): Promise<RoleMenuVisibility> {
  return getControlClient().get<RoleMenuVisibility>(
    `/v1/admin/menu-visibility/${encodeURIComponent(roleKey)}?workspace_id=${encodeURIComponent(workspaceId)}`
  );
}

export async function setRoleMenuVisibility(
  workspaceId: string,
  roleKey: Role,
  items: Record<string, PermissionVisibility>
): Promise<RoleMenuVisibility> {
  const payload: RoleMenuVisibility = { role_key: roleKey, items };
  return getControlClient().request<RoleMenuVisibility>(
    `/v1/admin/menu-visibility/${encodeURIComponent(roleKey)}?workspace_id=${encodeURIComponent(workspaceId)}`,
    { method: "PUT", body: payload }
  );
}
