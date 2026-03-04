import type {
  ChangeSetCapability,
  DiffChangeType,
  InspectorTabKey,
  MessageRole,
  PermissionMode,
  ProjectKind,
  QueueState,
  RunControlAction,
  RunState,
  TraceDetailLevel
} from "./api-common";

export type Project = {
  id: string;
  workspace_id: string;
  name: string;
  repo_path: string;
  is_git: boolean;
  default_model_config_id?: string;
  token_threshold?: number;
  tokens_in_total?: number;
  tokens_out_total?: number;
  tokens_total?: number;
  default_mode?: PermissionMode;
  current_revision: number;
  created_at: string;
  updated_at: string;
};

export type ProjectConfig = {
  project_id: string;
  model_config_ids: string[];
  default_model_config_id: string | null;
  token_threshold?: number;
  model_token_thresholds?: Record<string, number>;
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
  default_mode: PermissionMode;
  model_config_id: string;
  rule_ids: string[];
  skill_ids: string[];
  mcp_ids: string[];
  base_revision: number;
  active_execution_id: string | null;
  tokens_in_total?: number;
  tokens_out_total?: number;
  tokens_total?: number;
  created_at: string;
  updated_at: string;
};

// Session/Run aliases are introduced as the v1 runtime canonical terms.
// Existing Conversation/Execution exports remain during the Desktop migration.
export type Session = Conversation;

export type ConversationMessage = {
  id: string;
  conversation_id: string;
  role: MessageRole;
  content: string;
  created_at: string;
  queue_index?: number;
  can_rollback?: boolean;
};

export type SessionMessage = ConversationMessage;

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
    state: RunState;
    queue_index: number;
    message_id: string;
    updated_at: string;
  }>;
  execution_ids: string[];
  created_at: string;
};

export type SessionSnapshot = ConversationSnapshot;

export type ConversationDetailResponse = {
  conversation: Conversation;
  messages: ConversationMessage[];
  executions: Execution[];
  snapshots: ConversationSnapshot[];
};

export type SessionDetailResponse = ConversationDetailResponse;

export type Execution = {
  id: string;
  workspace_id: string;
  conversation_id: string;
  message_id: string;
  state: RunState;
  mode: PermissionMode;
  model_id: string;
  mode_snapshot: PermissionMode;
  model_snapshot: {
    config_id?: string;
    vendor?: string;
    model_id: string;
    base_url?: string;
    base_url_key?: string;
    runtime?: {
      request_timeout_ms?: number;
    };
    params?: Record<string, unknown>;
  };
  resource_profile_snapshot?: {
    model_config_id?: string;
    model_id: string;
    rule_ids?: string[];
    skill_ids?: string[];
    mcp_ids?: string[];
    project_file_paths?: string[];
  };
  agent_config_snapshot?: {
    max_model_turns: number;
    show_process_trace: boolean;
    trace_detail_level: TraceDetailLevel;
  };
  tokens_in?: number;
  tokens_out?: number;
  project_revision_snapshot: number;
  queue_index: number;
  trace_id: string;
  created_at: string;
  updated_at: string;
};

export type Run = Execution;

export type RunEventType =
  | "message_received"
  | "user_prompt_submit"
  | "execution_started"
  | "thinking_delta"
  | "pre_tool_use"
  | "permission_request"
  | "tool_call"
  | "tool_result"
  | "post_tool_use"
  | "post_tool_use_failure"
  | "diff_generated"
  | "change_set_updated"
  | "change_set_committed"
  | "change_set_discarded"
  | "change_set_rolled_back"
  | "execution_stopped"
  | "execution_done"
  | "execution_error"
  | "task_graph_configured"
  | "task_dependencies_updated"
  | "task_retry_policy_updated"
  | "task_artifact_emitted"
  | "task_failed"
  | "task_started"
  | "task_completed"
  | "task_cancelled";

export type ExecutionEvent = {
  event_id: string;
  execution_id: string;
  conversation_id: string;
  trace_id: string;
  sequence: number;
  queue_index: number;
  type: RunEventType;
  timestamp: string;
  payload: Record<string, unknown>;
};

export type RunLifecycleEvent = ExecutionEvent;

export type StreamRunEventType =
  | "run_queued"
  | "run_started"
  | "run_output_delta"
  | "run_approval_needed"
  | "run_completed"
  | "run_failed"
  | "run_cancelled";

export type RunEvent = {
  type: StreamRunEventType;
  session_id: string;
  run_id: string;
  sequence: number;
  timestamp: string;
  payload: Record<string, unknown>;
  event_id?: string;
};

export type ConversationStreamEvent = ExecutionEvent | RunEvent;

export type SessionStreamEvent = ConversationStreamEvent;

export type RunControlRequest = {
  action: RunControlAction;
  answer?: {
    question_id: string;
    selected_option_id?: string;
    text?: string;
  };
};

export type RunControlResponse = {
  ok: true;
  run_id: string;
  state: string;
  previous_state: string;
};

export type DiffItem = {
  id: string;
  path: string;
  change_type: DiffChangeType;
  summary: string;
  added_lines?: number;
  deleted_lines?: number;
};

export type ChangeEntry = {
  entry_id: string;
  message_id: string;
  execution_id: string;
  path: string;
  change_type: DiffChangeType;
  summary: string;
  added_lines?: number;
  deleted_lines?: number;
  before_blob?: string;
  after_blob?: string;
  created_at: string;
};

export type CommitSuggestion = {
  message: string;
};

export type CheckpointSummary = {
  checkpoint_id: string;
  message: string;
  created_at: string;
  git_commit_id?: string;
  entries_digest?: string;
};

export type ConversationChangeSet = {
  change_set_id: string;
  conversation_id: string;
  project_kind: ProjectKind;
  entries: ChangeEntry[];
  file_count: number;
  added_lines: number;
  deleted_lines: number;
  capability: ChangeSetCapability;
  suggested_message: CommitSuggestion;
  last_committed_checkpoint?: CheckpointSummary;
};

export type SessionChangeSet = ConversationChangeSet;

export type ChangeSetCommitRequest = {
  message: string;
  expected_change_set_id: string;
};

export type ChangeSetDiscardRequest = {
  expected_change_set_id: string;
};

export type ChangeSetCommitResponse = {
  ok: true;
  checkpoint: CheckpointSummary;
};

export type ExecutionFilesExportResponse = {
  file_name: string;
  archive_base64: string;
};

export type ComposerResourceType = "model" | "rule" | "skill" | "mcp" | "file";

export type ComposerResourceSelection = {
  type: ComposerResourceType;
  id: string;
};

export type ComposerCommandCatalogItem = {
  name: string;
  description: string;
  kind: "control" | "prompt";
};

export type ComposerResourceCatalogItem = {
  type: ComposerResourceType;
  id: string;
  name: string;
};

export type ComposerCatalog = {
  revision: string;
  commands: ComposerCommandCatalogItem[];
  resources: ComposerResourceCatalogItem[];
};

export type ComposerSuggestion = {
  kind: "command" | "resource_type" | "resource";
  label: string;
  detail?: string;
  insert_text: string;
  replace_start: number;
  replace_end: number;
};

export type ComposerSuggestRequest = {
  draft: string;
  cursor: number;
  limit?: number;
  catalog_revision?: string;
};

export type ComposerSuggestResponse = {
  revision: string;
  suggestions: ComposerSuggestion[];
};

export type ComposerSubmitRequest = {
  raw_input: string;
  mode: PermissionMode;
  model_config_id?: string;
  selected_resources?: ComposerResourceSelection[];
  catalog_revision?: string;
};

export type ComposerSubmitResponse =
  | {
    kind: "command_result";
    command_result: {
      command: string;
      output: string;
    };
  }
  | {
    kind: "execution_enqueued";
    execution: Execution;
    queue_state: QueueState;
    queue_index: number;
  };

export type ConversationRuntime = {
  mode: PermissionMode;
  modelId: string;
  ruleIds: string[];
  skillIds: string[];
  mcpIds: string[];
  projectKind: ProjectKind;
  draft: string;
  messages: ConversationMessage[];
  executions: Execution[];
  diff: DiffItem[];
  changeSet?: ConversationChangeSet | null;
  inspectorTab: InspectorTabKey;
  diffCapability: ChangeSetCapability;
};

export type SessionRuntime = ConversationRuntime;
