import type {
  ConversationMode,
  DiffCapability,
  DiffChangeType,
  ExecutionState,
  InspectorTabKey,
  MessageRole,
  QueueState
} from "@/shared/types/api-common";

export type Project = {
  id: string;
  workspace_id: string;
  name: string;
  repo_path: string;
  is_git: boolean;
  default_model_id?: string;
  default_mode?: ConversationMode;
  current_revision: number;
  created_at: string;
  updated_at: string;
};

export type ProjectConfig = {
  project_id: string;
  model_ids: string[];
  default_model_id: string | null;
  rule_ids: string[];
  skill_ids: string[];
  mcp_ids: string[];
  updated_at: string;
};

export type Conversation = {
  id: string;
  workspace_id: string;
  project_id: string;
  name: string;
  queue_state: QueueState;
  default_mode: ConversationMode;
  model_id: string;
  base_revision: number;
  active_execution_id: string | null;
  created_at: string;
  updated_at: string;
};

export type ConversationMessage = {
  id: string;
  conversation_id: string;
  role: MessageRole;
  content: string;
  created_at: string;
  queue_index?: number;
  can_rollback?: boolean;
};

export type ConversationSnapshot = {
  id: string;
  conversation_id: string;
  rollback_point_message_id: string;
  queue_state: QueueState;
  worktree_ref: string | null;
  inspector_state: {
    tab: InspectorTabKey;
  };
  messages: ConversationMessage[];
  execution_snapshots?: Array<{
    id: string;
    state: ExecutionState;
    queue_index: number;
    message_id: string;
    updated_at: string;
  }>;
  execution_ids: string[];
  created_at: string;
};

export type ConversationDetailResponse = {
  conversation: Conversation;
  messages: ConversationMessage[];
  executions: Execution[];
  snapshots: ConversationSnapshot[];
};

export type Execution = {
  id: string;
  workspace_id: string;
  conversation_id: string;
  message_id: string;
  state: ExecutionState;
  mode: ConversationMode;
  model_id: string;
  mode_snapshot: ConversationMode;
  model_snapshot: {
    config_id?: string;
    vendor?: string;
    model_id: string;
    base_url?: string;
    timeout_ms?: number;
    params?: Record<string, unknown>;
  };
  project_revision_snapshot: number;
  queue_index: number;
  trace_id: string;
  created_at: string;
  updated_at: string;
};

export type ExecutionEventType =
  | "message_received"
  | "execution_started"
  | "thinking_delta"
  | "tool_call"
  | "tool_result"
  | "confirmation_required"
  | "confirmation_resolved"
  | "diff_generated"
  | "execution_stopped"
  | "execution_done"
  | "execution_error";

export type ExecutionEvent = {
  event_id: string;
  execution_id: string;
  conversation_id: string;
  trace_id: string;
  sequence: number;
  queue_index: number;
  type: ExecutionEventType;
  timestamp: string;
  payload: Record<string, unknown>;
};

export type DiffItem = {
  id: string;
  path: string;
  change_type: DiffChangeType;
  summary: string;
};

export type ExecutionCreateRequest = {
  content: string;
  mode: ConversationMode;
  model_id: string;
};

export type ExecutionCreateResponse = {
  execution: Execution;
  queue_state: QueueState;
  queue_index: number;
};

export type ConversationRuntime = {
  mode: ConversationMode;
  modelId: string;
  draft: string;
  messages: ConversationMessage[];
  executions: Execution[];
  diff: DiffItem[];
  inspectorTab: InspectorTabKey;
  diffCapability: DiffCapability;
};
