import type { Role } from "@/shared/types/api-common";

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
