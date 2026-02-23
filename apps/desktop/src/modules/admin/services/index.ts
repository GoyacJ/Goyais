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
      const existing = mockData.users.find(
        (user) => user.workspace_id === workspaceId && user.username === input.username
      );
      if (existing) {
        existing.display_name = input.display_name;
        existing.role = input.role;
        return existing;
      }

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

export async function removeAdminUser(userId: string): Promise<void> {
  return withApiFallback(
    "admin.removeUser",
    async () => {
      await getControlClient().request<void>(`/v1/admin/users/${userId}`, { method: "DELETE" });
    },
    () => {
      mockData.users = mockData.users.filter((item) => item.id !== userId);
    }
  );
}

export async function setAdminUserEnabled(userId: string, enabled: boolean): Promise<AdminUser> {
  return withApiFallback(
    "admin.setUserEnabled",
    () => getControlClient().request<AdminUser>(`/v1/admin/users/${userId}`, { method: "PATCH", body: { enabled } }),
    () => {
      const target = mockData.users.find((item) => item.id === userId);
      if (!target) {
        throw new Error("User not found");
      }
      target.enabled = enabled;
      return target;
    }
  );
}

export async function setAdminUserRole(userId: string, role: Role): Promise<AdminUser> {
  return withApiFallback(
    "admin.setUserRole",
    () => getControlClient().request<AdminUser>(`/v1/admin/users/${userId}`, { method: "PATCH", body: { role } }),
    () => {
      const target = mockData.users.find((item) => item.id === userId);
      if (!target) {
        throw new Error("User not found");
      }
      target.role = role;
      return target;
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

export async function upsertAdminRole(input: AdminRole): Promise<AdminRole> {
  return withApiFallback(
    "admin.upsertRole",
    () => getControlClient().post<AdminRole>("/v1/admin/roles", input),
    () => {
      const existingIndex = mockData.roles.findIndex((item) => item.key === input.key);
      if (existingIndex >= 0) {
        mockData.roles[existingIndex] = { ...input, permissions: [...input.permissions] };
        return mockData.roles[existingIndex];
      }

      const created: AdminRole = {
        key: input.key,
        name: input.name,
        permissions: [...input.permissions],
        enabled: input.enabled
      };
      mockData.roles.push(created);
      return created;
    }
  );
}

export async function removeAdminRole(roleKey: Role): Promise<void> {
  return withApiFallback(
    "admin.removeRole",
    async () => {
      await getControlClient().request<void>(`/v1/admin/roles/${roleKey}`, { method: "DELETE" });
    },
    () => {
      mockData.roles = mockData.roles.filter((item) => item.key !== roleKey);
    }
  );
}

export async function setAdminRoleEnabled(roleKey: Role, enabled: boolean): Promise<AdminRole> {
  return withApiFallback(
    "admin.setRoleEnabled",
    () => getControlClient().request<AdminRole>(`/v1/admin/roles/${roleKey}`, { method: "PATCH", body: { enabled } }),
    () => {
      const target = mockData.roles.find((item) => item.key === roleKey);
      if (!target) {
        throw new Error("Role not found");
      }
      target.enabled = enabled;
      return target;
    }
  );
}

export async function listAuditEvents(workspaceId: string): Promise<AdminAuditEvent[]> {
  return withApiFallback(
    "admin.listAudit",
    () => getControlClient().get<AdminAuditEvent[]>(`/v1/admin/audit?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => [...mockData.auditEvents]
  );
}
