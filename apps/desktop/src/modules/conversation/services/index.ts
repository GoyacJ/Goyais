import { getControlClient } from "@/shared/services/clients";
import { getControlHubBaseUrl } from "@/shared/runtime";
import { connectConversationEvents } from "@/shared/services/sseClient";
import type {
  OpenAPIContractComponents,
  ChangeSetCommitRequest,
  ChangeSetCommitResponse,
  ChangeSetCapability,
  ChangeSetDiscardRequest,
  ComposerCatalog,
  ComposerSubmitRequest,
  ComposerSubmitResponse,
  ComposerSuggestRequest,
  ComposerSuggestResponse,
  Run,
  RunControlAction,
  RunControlResponse,
  RunFilesExportResponse,
  Session,
  SessionChangeSet,
  SessionDetailResponse,
  SessionStreamEvent,
  SessionSubmitResponse,
} from "@/shared/types/api";

type ConversationServiceOptions = {
  token?: string;
};

export type SessionServiceOptions = ConversationServiceOptions;

type AgentGraph = OpenAPIContractComponents["schemas"]["AgentGraph"];
type TaskNode = OpenAPIContractComponents["schemas"]["TaskNode"];
type TaskState = OpenAPIContractComponents["schemas"]["TaskState"];
type RunTaskListResponse = OpenAPIContractComponents["schemas"]["RunTaskListResponse"];
type TaskControlAction = OpenAPIContractComponents["schemas"]["TaskControlRequest"]["action"];
type TaskControlResponse = OpenAPIContractComponents["schemas"]["TaskControlResponse"];

export type ConversationRunTaskGraph = AgentGraph;
export type ConversationRunTaskNode = TaskNode;
export type ConversationRunTaskState = TaskState;
export type ConversationRunTaskListResponse = RunTaskListResponse;
export type ConversationRunTaskControlAction = TaskControlAction;
export type ConversationRunTaskControlResponse = TaskControlResponse;
export type SessionRunTaskGraph = ConversationRunTaskGraph;
export type SessionRunTaskNode = ConversationRunTaskNode;
export type SessionRunTaskState = ConversationRunTaskState;
export type SessionRunTaskListResponse = ConversationRunTaskListResponse;
export type SessionRunTaskControlAction = ConversationRunTaskControlAction;
export type SessionRunTaskControlResponse = ConversationRunTaskControlResponse;

export function streamConversationEvents(
  conversationId: string,
  options: {
    token?: string;
    initialLastEventId?: string;
    onEvent: (event: SessionStreamEvent) => void;
    onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
    onError: (error: Error) => void;
  }
) {
  const url = `${getHubBaseUrl()}/v1/sessions/${conversationId}/events`;
  return connectConversationEvents(url, {
    token: options.token,
    initialLastEventId: options.initialLastEventId,
    onEvent: options.onEvent,
    onStatusChange: options.onStatusChange,
    onError: options.onError
  });
}

export function streamSessionEvents(
  sessionId: string,
  options: {
    token?: string;
    initialLastEventId?: string;
    onEvent: (event: SessionStreamEvent) => void;
    onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
    onError: (error: Error) => void;
  }
) {
  return streamConversationEvents(sessionId, options);
}

export async function getComposerCatalog(conversationId: string): Promise<ComposerCatalog> {
  return getControlClient().get<ComposerCatalog>(`/v1/sessions/${conversationId}/input/catalog`);
}

export async function getSessionComposerCatalog(sessionId: string): Promise<ComposerCatalog> {
  return getComposerCatalog(sessionId);
}

export async function suggestComposerInput(
  conversationId: string,
  input: ComposerSuggestRequest
): Promise<ComposerSuggestResponse> {
  return getControlClient().post<ComposerSuggestResponse>(`/v1/sessions/${conversationId}/input/suggest`, input);
}

export async function suggestSessionInput(
  sessionId: string,
  input: ComposerSuggestRequest
): Promise<ComposerSuggestResponse> {
  return suggestComposerInput(sessionId, input);
}

export async function submitComposerInput(
  session: Session,
  input: ComposerSubmitRequest
): Promise<SessionSubmitResponse> {
  const response = await getControlClient().post<ComposerSubmitResponse>(`/v1/sessions/${session.id}/runs`, input);
  return normalizeSessionSubmitResponse(response);
}

export async function submitSessionInput(
  session: Session,
  input: ComposerSubmitRequest
): Promise<SessionSubmitResponse> {
  return submitComposerInput(session, input);
}

export async function getConversationDetail(
  conversationId: string,
  options: ConversationServiceOptions = {}
): Promise<SessionDetailResponse> {
  const detail = await getControlClient().get<SessionDetailWirePayload>(`/v1/sessions/${conversationId}`, { token: options.token });
  return normalizeSessionDetailResponse(detail);
}

export async function getSessionDetail(
  sessionId: string,
  options: SessionServiceOptions = {}
): Promise<SessionDetailResponse> {
  return getConversationDetail(sessionId, options);
}

export async function cancelExecution(conversationId: string, executionId: string): Promise<void> {
  void executionId;
  await getControlClient().post<void>(`/v1/sessions/${conversationId}/stop`);
}

export async function cancelSessionRun(sessionId: string, runId: string): Promise<void> {
  await cancelExecution(sessionId, runId);
}

export async function controlExecutionRun(
  executionId: string,
  action: RunControlAction,
  answer?: {
    question_id: string;
    selected_option_id?: string;
    text?: string;
  }
): Promise<RunControlResponse> {
  return getControlClient().post<RunControlResponse>(`/v1/runs/${executionId}/control`, {
    action,
    ...(answer ? { answer } : {})
  });
}

export async function controlRun(
  runId: string,
  action: RunControlAction,
  answer?: {
    question_id: string;
    selected_option_id?: string;
    text?: string;
  }
): Promise<RunControlResponse> {
  return controlExecutionRun(runId, action, answer);
}

export async function getRunTaskGraph(runId: string): Promise<AgentGraph> {
  return getControlClient().get<AgentGraph>(`/v1/runs/${runId}/graph`);
}

export async function listRunTasks(
  runId: string,
  options: {
    state?: TaskState;
    cursor?: string;
    limit?: number;
  } = {}
): Promise<RunTaskListResponse> {
  const query = new URLSearchParams();
  if (options.state) {
    query.set("state", options.state);
  }
  if (options.cursor) {
    query.set("cursor", options.cursor);
  }
  if (typeof options.limit === "number" && Number.isFinite(options.limit)) {
    query.set("limit", String(Math.max(1, Math.trunc(options.limit))));
  }
  const suffix = query.toString();
  const path = suffix === "" ? `/v1/runs/${runId}/tasks` : `/v1/runs/${runId}/tasks?${suffix}`;
  return getControlClient().get<RunTaskListResponse>(path);
}

export async function getRunTaskById(runId: string, taskId: string): Promise<TaskNode> {
  return getControlClient().get<TaskNode>(`/v1/runs/${runId}/tasks/${taskId}`);
}

export async function controlRunTask(
  runId: string,
  taskId: string,
  action: TaskControlAction,
  reason?: string
): Promise<TaskControlResponse> {
  return getControlClient().post<TaskControlResponse>(`/v1/runs/${runId}/tasks/${taskId}/control`, {
    action,
    ...(reason ? { reason } : {})
  });
}

export async function rollbackExecution(conversationId: string, messageId: string): Promise<void> {
  await getControlClient().post<void>(`/v1/sessions/${conversationId}/rollback`, {
    message_id: messageId
  });
}

export async function rollbackSessionToMessage(sessionId: string, messageId: string): Promise<void> {
  await rollbackExecution(sessionId, messageId);
}

export async function getConversationChangeSet(conversationId: string): Promise<SessionChangeSet> {
  return getControlClient().get<SessionChangeSet>(`/v1/sessions/${conversationId}/changeset`);
}

export async function getSessionChangeSet(sessionId: string): Promise<SessionChangeSet> {
  return getConversationChangeSet(sessionId);
}

export async function commitConversationChangeSet(
  conversationId: string,
  input: ChangeSetCommitRequest
): Promise<ChangeSetCommitResponse> {
  return getControlClient().post<ChangeSetCommitResponse>(`/v1/sessions/${conversationId}/changeset/commit`, input);
}

export async function commitSessionChangeSet(
  sessionId: string,
  input: ChangeSetCommitRequest
): Promise<ChangeSetCommitResponse> {
  return commitConversationChangeSet(sessionId, input);
}

export async function discardConversationChangeSet(
  conversationId: string,
  input: ChangeSetDiscardRequest
): Promise<void> {
  await getControlClient().post<void>(`/v1/sessions/${conversationId}/changeset/discard`, input);
}

export async function discardSessionChangeSet(
  sessionId: string,
  input: ChangeSetDiscardRequest
): Promise<void> {
  await discardConversationChangeSet(sessionId, input);
}

export async function exportConversationChangeSet(conversationId: string): Promise<RunFilesExportResponse> {
  return getControlClient().post<RunFilesExportResponse>(`/v1/sessions/${conversationId}/changeset/export`, {});
}

export async function exportSessionChangeSet(sessionId: string): Promise<RunFilesExportResponse> {
  return exportConversationChangeSet(sessionId);
}

export function resolveDiffCapability(_isGitProject: boolean): ChangeSetCapability {
  return {
    can_commit: true,
    can_discard: true,
    can_export: true,
    can_export_patch: true
  };
}

type SessionDetailWirePayload = {
  session?: Session;
  conversation?: Session;
  messages: SessionDetailResponse["messages"];
  runs?: Run[];
  executions?: Run[];
  snapshots: SessionDetailResponse["snapshots"];
};

function normalizeSessionDetailResponse(detail: SessionDetailWirePayload): SessionDetailResponse {
  const session = detail.session ?? detail.conversation;
  if (!session) {
    throw new Error("invalid session detail payload: session is required");
  }
  const runs = detail.runs ?? detail.executions ?? [];
  const normalized: SessionDetailResponse = {
    session,
    messages: detail.messages,
    runs,
    snapshots: detail.snapshots
  };
  if (detail.conversation) {
    normalized.conversation = detail.conversation;
  }
  if (detail.executions) {
    normalized.executions = detail.executions;
  }
  return normalized;
}

function normalizeSessionSubmitResponse(response: ComposerSubmitResponse | SessionSubmitResponse): SessionSubmitResponse {
  if (response.kind === "command_result") {
    return response;
  }

  if (response.kind === "run_enqueued" && "run" in response) {
    return response;
  }

  return {
    kind: "run_enqueued",
    run: response.execution,
    queue_state: response.queue_state,
    queue_index: response.queue_index
  };
}

function getHubBaseUrl(): string {
  return getControlHubBaseUrl();
}
