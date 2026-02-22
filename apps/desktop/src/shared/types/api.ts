export type WorkspaceMode = "local" | "remote";
export type AuthMode = "disabled" | "password_or_token" | "token_only";
export type Role = "viewer" | "developer" | "approver" | "admin";

export type Workspace = {
  id: string;
  name: string;
  mode: WorkspaceMode;
  hub_url: string | null;
  is_default_local: boolean;
  created_at: string;
  login_disabled: boolean;
  auth_mode: AuthMode;
};

export type ListEnvelope<T> = {
  items: T[];
  next_cursor: string | null;
};

export type CreateWorkspaceRequest = {
  name: string;
  hub_url: string;
  login_disabled?: boolean;
  auth_mode?: Exclude<AuthMode, "disabled">;
};

export type LoginRequest = {
  workspace_id: string;
  username?: string;
  password?: string;
  token?: string;
};

export type LoginResponse = {
  access_token: string;
  token_type: "bearer";
  expires_in?: number;
};

export type Capabilities = {
  admin_console: boolean;
  resource_write: boolean;
  execution_control: boolean;
};

export type Me = {
  user_id: string;
  display_name: string;
  workspace_id: string;
  role: Role;
  capabilities: Capabilities;
};

export type StandardError = {
  code: string;
  message: string;
  details: Record<string, unknown>;
  trace_id: string;
};
