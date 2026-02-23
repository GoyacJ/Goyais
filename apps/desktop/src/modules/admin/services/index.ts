import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type { AdminAuditEvent, AdminRole, AdminUser, ListEnvelope, Role } from "@/shared/types/api";

export async function listAdminUsers(workspaceId: string): Promise<ListEnvelope<AdminUser>> {
  return withApiFallback(
    "admin.listUsers",
    () => getControlClient().get<ListEnvelope<AdminUser>>(`/v1/admin/users?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => ({
      items: mockData.users.filter((user) => user.workspace_id === workspaceId),
      next_cursor: null
    })
  );
}

export async function upsertAdminUser(workspaceId: string, input: { username: string; display_name: string; role: Role }): Promise<AdminUser> {
  return withApiFallback(
    "admin.upsertUser",
    () => getControlClient().post<AdminUser>("/v1/admin/users", { workspace_id: workspaceId, ...input }),
    () => {
      const created: AdminUser = {
        id: createMockId("u"),
        workspace_id: workspaceId,
        username: input.username,
        display_name: input.display_name,
        role: input.role,
        enabled: true,
        created_at: new Date().toISOString()
      };
      mockData.users.push(created);
      return created;
    }
  );
}

export async function listAdminRoles(): Promise<AdminRole[]> {
  return withApiFallback(
    "admin.listRoles",
    () => getControlClient().get<AdminRole[]>("/v1/admin/roles"),
    () => [...mockData.roles]
  );
}

export async function listAuditEvents(workspaceId: string): Promise<AdminAuditEvent[]> {
  return withApiFallback(
    "admin.listAudit",
    () => getControlClient().get<AdminAuditEvent[]>(`/v1/admin/audit?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => [...mockData.auditEvents]
  );
}
