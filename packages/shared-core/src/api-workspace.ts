import type { AuthMode, ConnectionStatus, ConversationStatus, Role, TraceDetailLevel, WorkspaceMode } from "./api-common";

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

export type WorkspaceConnection = {
  workspace_id: string;
  hub_url: string;
  username: string;
  connection_status: ConnectionStatus;
  connected_at: string;
  access_token?: string;
};

export type WorkspaceConnectionResult = {
  workspace: Workspace;
  connection: WorkspaceConnection;
  access_token?: string;
};

export type WorkspaceStatusResponse = {
  workspace_id: string;
  conversation_id?: string;
  conversation_status: ConversationStatus;
  hub_url: string;
  connection_status: ConnectionStatus;
  user_display_name: string;
  updated_at: string;
};

export type WorkspaceAgentConfig = {
  workspace_id: string;
  execution: {
    max_model_turns: number;
  };
  display: {
    show_process_trace: boolean;
    trace_detail_level: TraceDetailLevel;
  };
  updated_at: string;
};

export type CreateWorkspaceRequest = {
  name?: string;
  hub_url: string;
  username: string;
  password: string;
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
  refresh_token?: string;
  token_type: "bearer";
  expires_in?: number;
};

export type RefreshRequest = {
  refresh_token: string;
};

export type LogoutRequest = {
  access_token?: string;
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
