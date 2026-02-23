import { reactive } from "vue";

import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type {
  ABACPolicy,
  AdminAuditEvent,
  AdminMenu,
  AdminPermission,
  AdminRole,
  AdminUser,
  PermissionVisibility,
  Role
} from "@/shared/types/api";

export type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

export type AdminState = {
  users: AdminUser[];
  usersPage: CursorPageState;
  roles: AdminRole[];
  audits: AdminAuditEvent[];
  auditsPage: CursorPageState;
  menuNodes: AdminMenu[];
  permissionItems: AdminPermission[];
  abacPolicies: ABACPolicy[];
  activeRoleKey: Role;
  roleMenuVisibility: Record<string, PermissionVisibility>;
  loading: boolean;
  error: string;
};

const initialState: AdminState = {
  users: [],
  usersPage: createInitialPageState(),
  roles: [],
  audits: [],
  auditsPage: createInitialPageState(),
  menuNodes: [],
  permissionItems: [],
  abacPolicies: [],
  activeRoleKey: "admin",
  roleMenuVisibility: {},
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
  adminStore.menuNodes = [];
  adminStore.permissionItems = [];
  adminStore.abacPolicies = [];
  adminStore.activeRoleKey = "admin";
  adminStore.roleMenuVisibility = {};
  adminStore.loading = false;
  adminStore.error = "";
}

export function applyUserUpdate(user: AdminUser): void {
  const index = adminStore.users.findIndex((item) => item.id === user.id);
  if (index >= 0) {
    adminStore.users[index] = user;
  }
}

export function upsertPermission(permission: AdminPermission): void {
  const index = adminStore.permissionItems.findIndex((item) => item.key === permission.key);
  if (index >= 0) {
    adminStore.permissionItems[index] = permission;
  } else {
    adminStore.permissionItems.push(permission);
  }
}

export function upsertMenu(menu: AdminMenu): void {
  const index = adminStore.menuNodes.findIndex((item) => item.key === menu.key);
  if (index >= 0) {
    adminStore.menuNodes[index] = menu;
  } else {
    adminStore.menuNodes.push(menu);
  }
}

export function upsertPolicy(policy: ABACPolicy): void {
  const index = adminStore.abacPolicies.findIndex((item) => item.id === policy.id);
  if (index >= 0) {
    adminStore.abacPolicies[index] = policy;
  } else {
    adminStore.abacPolicies.push(policy);
  }
}

export function getRemoteWorkspaceId(): string {
  const workspace = getCurrentWorkspace();
  if (!workspace || workspace.mode !== "remote") {
    return "";
  }
  return workspace.id;
}

export function createInitialPageState(limit = 20): CursorPageState {
  return {
    limit,
    currentCursor: null,
    backStack: [],
    nextCursor: null,
    loading: false
  };
}
