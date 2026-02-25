import { getControlClient } from "@/shared/services/clients";
import type { AdminAuditEvent, AdminRole, AdminUser, ListEnvelope, PaginationQuery, Role } from "@/shared/types/api";

export async function listAdminUsers(workspaceId: string, query: PaginationQuery = {}): Promise<ListEnvelope<AdminUser>> {
  const search = buildPaginationSearch({ ...query, workspace_id: workspaceId });
  return getControlClient().get<ListEnvelope<AdminUser>>(`/v1/admin/users${search}`);
}

export async function upsertAdminUser(
  workspaceId: string,
  input: { username: string; display_name: string; role: Role }
): Promise<AdminUser> {
  return getControlClient().post<AdminUser>("/v1/admin/users", { workspace_id: workspaceId, ...input });
}

export async function removeAdminUser(userId: string): Promise<void> {
  await getControlClient().request<void>(`/v1/admin/users/${userId}`, { method: "DELETE" });
}

export async function setAdminUserEnabled(userId: string, enabled: boolean): Promise<AdminUser> {
  return getControlClient().request<AdminUser>(`/v1/admin/users/${userId}`, { method: "PATCH", body: { enabled } });
}

export async function setAdminUserRole(userId: string, role: Role): Promise<AdminUser> {
  return getControlClient().request<AdminUser>(`/v1/admin/users/${userId}`, { method: "PATCH", body: { role } });
}

export async function listAdminRoles(workspaceId: string): Promise<AdminRole[]> {
  return getControlClient().get<AdminRole[]>(`/v1/admin/roles?workspace_id=${encodeURIComponent(workspaceId)}`);
}

export async function upsertAdminRole(workspaceId: string, input: AdminRole): Promise<AdminRole> {
  return getControlClient().post<AdminRole>(`/v1/admin/roles?workspace_id=${encodeURIComponent(workspaceId)}`, input);
}

export async function removeAdminRole(workspaceId: string, roleKey: Role): Promise<void> {
  await getControlClient().request<void>(`/v1/admin/roles/${roleKey}?workspace_id=${encodeURIComponent(workspaceId)}`, {
    method: "DELETE"
  });
}

export async function setAdminRoleEnabled(workspaceId: string, roleKey: Role, enabled: boolean): Promise<AdminRole> {
  return getControlClient().request<AdminRole>(`/v1/admin/roles/${roleKey}?workspace_id=${encodeURIComponent(workspaceId)}`, {
    method: "PATCH",
    body: { enabled }
  });
}

export async function listAuditEvents(workspaceId: string, query: PaginationQuery = {}): Promise<ListEnvelope<AdminAuditEvent>> {
  const search = buildPaginationSearch({ ...query, workspace_id: workspaceId });
  return getControlClient().get<ListEnvelope<AdminAuditEvent>>(`/v1/admin/audit${search}`);
}

function buildPaginationSearch(query: PaginationQuery & { workspace_id?: string }): string {
  const params = new URLSearchParams();
  if (query.workspace_id) {
    params.set("workspace_id", query.workspace_id);
  }
  if (query.cursor) {
    params.set("cursor", query.cursor);
  }
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}
