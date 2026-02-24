import type { ABACEffect, PermissionVisibility, Role } from "@/shared/types/api-common";

export type AdminUser = {
  id: string;
  workspace_id: string;
  username: string;
  display_name: string;
  role: Role;
  enabled: boolean;
  created_at: string;
};

export type AdminRole = {
  key: Role;
  name: string;
  permissions: string[];
  enabled: boolean;
};

export type AdminAuditEvent = {
  id: string;
  actor: string;
  action: string;
  resource: string;
  result: "success" | "denied" | "failed";
  trace_id: string;
  timestamp: string;
};

export type PermissionSnapshot = {
  role: Role;
  permissions: string[];
  menu_visibility: Record<string, PermissionVisibility>;
  action_visibility: Record<string, PermissionVisibility>;
  policy_version: string;
  generated_at: string;
};

export type AdminPermission = {
  key: string;
  label: string;
  enabled: boolean;
};

export type AdminMenu = {
  key: string;
  label: string;
  enabled: boolean;
};

export type RoleMenuVisibility = {
  role_key: Role;
  items: Record<string, PermissionVisibility>;
};

export type ABACPolicy = {
  id: string;
  workspace_id: string;
  name: string;
  effect: ABACEffect;
  priority: number;
  enabled: boolean;
  subject_expr: Record<string, unknown>;
  resource_expr: Record<string, unknown>;
  action_expr: Record<string, unknown>;
  context_expr: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
};
