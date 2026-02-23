import { reactive } from "vue";

import {
  listAdminRoles,
  listAdminUsers,
  listAuditEvents,
  upsertAdminUser
} from "@/modules/admin/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { AdminAuditEvent, AdminRole, AdminUser, Role } from "@/shared/types/api";

type AdminState = {
  users: AdminUser[];
  roles: AdminRole[];
  audits: AdminAuditEvent[];
  loading: boolean;
  error: string;
};

const initialState: AdminState = {
  users: [],
  roles: [],
  audits: [],
  loading: false,
  error: ""
};

export const adminStore = reactive<AdminState>({ ...initialState });

export function resetAdminStore(): void {
  adminStore.users = [];
  adminStore.roles = [];
  adminStore.audits = [];
  adminStore.loading = false;
  adminStore.error = "";
}

export async function refreshAdminData(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || workspace.mode !== "remote") {
    return;
  }

  adminStore.loading = true;
  adminStore.error = "";

  try {
    const [users, roles, audits] = await Promise.all([
      listAdminUsers(workspace.id),
      listAdminRoles(),
      listAuditEvents(workspace.id)
    ]);

    adminStore.users = users.items;
    adminStore.roles = roles;
    adminStore.audits = audits;
  } catch (error) {
    adminStore.error = toDisplayError(error);
  } finally {
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
  } catch (error) {
    adminStore.error = toDisplayError(error);
  }
}
