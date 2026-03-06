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

export type Session = {
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
  active_run_id: string | null;
  tokens_in_total?: number;
  tokens_out_total?: number;
  tokens_total?: number;
  created_at: string;
  updated_at: string;
};

export type SessionMessage = {
  id: string;
  session_id: string;
  role: MessageRole;
  content: string;
  created_at: string;
  queue_index?: number;
  can_rollback?: boolean;
};

export type SessionSnapshot = {
  id: string;
  session_id: string;
  rollback_point_message_id: string;
  queue_state: QueueState;
  worktree_ref: string | null;
  inspector_state: {
    tab: InspectorTabKey;
  };
  messages: SessionMessage[];
  execution_snapshots?: Array<{
    id: string;
    state: RunState;
    queue_index: number;
    message_id: string;
    updated_at: string;
  }>;
  run_ids: string[];
  created_at: string;
};

export type SessionDetailResponse = {
  session: Session;
  messages: SessionMessage[];
  runs: Run[];
  snapshots: SessionSnapshot[];
};

export type RunCapabilityDescriptorSnapshot = {
  id: string;
  kind: "builtin_tool" | "mcp_tool" | "mcp_prompt" | "skill" | "slash_command" | "subagent" | "output_style";
  name: string;
  description: string;
  source: string;
  scope: "system" | "workspace" | "project" | "user" | "local" | "plugin" | "managed";
  version: string;
  input_schema?: Record<string, unknown>;
  risk_level: string;
  read_only: boolean;
  concurrency_safe: boolean;
  requires_permissions: boolean;
  visibility_policy: "always_loaded" | "searchable";
  prompt_budget_cost: number;
};

export type RunMCPServerSnapshot = {
  name: string;
  transport: string;
  endpoint?: string;
  command?: string;
  env?: Record<string, string>;
  tools?: string[];
};

export type Run = {
  id: string;
  workspace_id: string;
  session_id: string;
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
    rules_dsl?: string;
    mcp_servers?: RunMCPServerSnapshot[];
    always_loaded_capabilities?: RunCapabilityDescriptorSnapshot[];
    searchable_capabilities?: RunCapabilityDescriptorSnapshot[];
  };
  agent_config_snapshot?: {
    max_model_turns: number;
    show_process_trace: boolean;
    trace_detail_level: TraceDetailLevel;
    default_mode: PermissionMode;
    builtin_tools?: string[];
    capability_budgets: {
      prompt_budget_chars: number;
      search_threshold_percent: number;
    };
    mcp_search: {
      enabled: boolean;
      result_limit: number;
    };
    output_style?: string;
    subagent_defaults: {
      max_turns: number;
      allowed_tools?: string[];
    };
    feature_flags: {
      enable_tool_search: boolean;
      enable_capability_graph: boolean;
    };
  };
  tokens_in?: number;
  tokens_out?: number;
  project_revision_snapshot: number;
  queue_index: number;
  trace_id: string;
  created_at: string;
  updated_at: string;
};

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

export type RunLifecycleEvent = {
  event_id: string;
  run_id: string;
  session_id: string;
  trace_id: string;
  sequence: number;
  queue_index: number;
  type: RunEventType;
  timestamp: string;
  payload: Record<string, unknown>;
};

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

export type SessionStreamEvent = RunLifecycleEvent | RunEvent;

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
  run_id: string;
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

export type SessionChangeSet = {
  change_set_id: string;
  session_id: string;
  project_kind: ProjectKind;
  entries: ChangeEntry[];
  file_count: number;
  added_lines: number;
  deleted_lines: number;
  capability: ChangeSetCapability;
  suggested_message: CommitSuggestion;
  last_committed_checkpoint?: CheckpointSummary;
};

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

export type RunFilesExportResponse = {
  file_name: string;
  archive_base64: string;
};

export type ComposerCapabilityKind = "model" | "rule" | "skill" | "mcp" | "file";

export type ComposerCommandCatalogItem = {
  name: string;
  description: string;
  kind: "control" | "prompt";
};

export type ComposerCapabilityCatalogItem = {
  id: string;
  kind: ComposerCapabilityKind;
  name: string;
  description?: string;
  source?: string;
  scope?: string;
};

export type ComposerCatalog = {
  revision: string;
  commands: ComposerCommandCatalogItem[];
  capabilities: ComposerCapabilityCatalogItem[];
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
  selected_capabilities?: string[];
  catalog_revision?: string;
};

export type ComposerSubmitResponse =
  | {
    kind: "command_result";
    command_result: {
      command: string;
      output: string;
    };
  };

export type SessionSubmitResponse =
  | {
    kind: "command_result";
    command_result: {
      command: string;
      output: string;
    };
  }
  | {
    kind: "run_enqueued";
    run: Run;
    queue_state: QueueState;
    queue_index: number;
  };

export type SessionRuntime = {
  mode: PermissionMode;
  modelId: string;
  ruleIds: string[];
  skillIds: string[];
  mcpIds: string[];
  projectKind: ProjectKind;
  draft: string;
  messages: SessionMessage[];
  runs: Run[];
  diff: DiffItem[];
  changeSet?: SessionChangeSet | null;
  inspectorTab: InspectorTabKey;
  diffCapability: ChangeSetCapability;
};
