import {
  cancelSessionRun,
  commitSessionChangeSet,
  controlRunTask,
  controlRun,
  discardSessionChangeSet,
  getComposerCatalog,
  getSessionChangeSet,
  getSessionDetail,
  getRunTaskById,
  getRunTaskGraph,
  rollbackSessionToMessage,
  listRunTasks,
  submitSessionInput
} from "@/modules/session/services";
import { createExecutionEvent } from "@/modules/session/store/events";
import {
  dedupeExecutions,
  restoreExecutionsFromSnapshot,
  upsertExecutionFromServer
} from "@/modules/session/store/executionRuntime";
import { buildEventDedupKey, rememberProcessedEvent } from "@/modules/session/store/executionEventIdempotency";
import {
  appendRuntimeEvent,
  sessionStore,
  createSessionSnapshot,
  ensureSessionRuntime,
  findSessionSnapshotForMessage,
  hydrateSessionRuntime,
  pushSessionSnapshot,
  setSessionChangeSet
} from "@/modules/session/store/state";
import {
  applyDiffUpdate,
  appendTerminalMessageFromEvent,
  updateExecutionTransition
} from "@/modules/session/store/executionEventHandlers";
import { toDisplayError } from "@/shared/services/errorMapper";
import { ApiError } from "@/shared/services/http";
import { createMockId } from "@/shared/utils/id";
import type { ComposerResourceSelection, RunLifecycleEvent, Session, SessionMessage } from "@/shared/types/api";
import type {
  ConversationRunTaskControlAction,
  ConversationRunTaskControlResponse,
  ConversationRunTaskGraph,
  ConversationRunTaskListResponse,
  ConversationRunTaskNode,
  ConversationRunTaskState
} from "@/modules/session/services";

export async function submitConversationMessage(
  conversation: Session,
  isGitProject: boolean,
  options: {
    catalogRevision?: string;
  } = {}
): Promise<void> {
  const runtime = ensureSessionRuntime(conversation, isGitProject);
  const content = runtime.draft.trim();
  if (content === "") {
    return;
  }
  if (runtime.modelId.trim() === "") {
    sessionStore.error = "当前项目未绑定可用模型，请先在项目配置中绑定模型";
    return;
  }

  runtime.draft = "";
  const queueIndex = runtime.executions.length;

  const userMessage: SessionMessage = {
    id: createMockId("msg"),
    session_id: conversation.id,
    role: "user",
    content,
    queue_index: queueIndex,
    can_rollback: true,
    created_at: new Date().toISOString()
  };
  runtime.messages.push(userMessage);
  pushSessionSnapshot(
    conversation.id,
    createSessionSnapshot(runtime, conversation.id, userMessage.id)
  );

  try {
    const baseInput = {
      raw_input: content,
      mode: runtime.mode,
      model_config_id: runtime.modelId.trim() || undefined,
      selected_resources: extractSelectedResources(content),
      catalog_revision: options.catalogRevision
    };
    let response;
    try {
      response = await submitSessionInput(conversation, baseInput);
    } catch (error) {
      if (isCatalogStaleError(error)) {
        const catalog = await getComposerCatalog(conversation.id);
        response = await submitSessionInput(conversation, {
          ...baseInput,
          catalog_revision: catalog.revision
        });
      } else {
        throw error;
      }
    }

    if (response.kind === "command_result") {
      runtime.messages.push({
        id: createMockId("msg"),
        session_id: conversation.id,
        role: "system",
        content: response.command_result.output,
        created_at: new Date().toISOString()
      });
      return;
    }

    const run = response.run;
    upsertExecutionFromServer(runtime, run);
    dedupeExecutions(runtime);
    appendRuntimeEvent(
      runtime,
      createExecutionEvent(conversation.id, run.id, response.queue_index, "message_received", {
        message_id: run.message_id,
        queue_state: response.queue_state
      })
    );
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    runtime.messages.push({
      id: createMockId("msg"),
      session_id: conversation.id,
      role: "system",
      content: sessionStore.error,
      created_at: new Date().toISOString()
    });
  }
}

function extractSelectedResources(rawInput: string): ComposerResourceSelection[] {
  const normalized = rawInput.trim();
  if (normalized === "") {
    return [];
  }
  const mentionPattern = /@(?<type>model|rule|skill|mcp|file):(?<id>[\w./-]+)/g;
  const seen = new Set<string>();
  const selections: ComposerResourceSelection[] = [];
  for (const match of normalized.matchAll(mentionPattern)) {
    const typeRaw = (match.groups?.type ?? "").trim();
    const id = (match.groups?.id ?? "").trim();
    if (typeRaw === "" || id === "") {
      continue;
    }
    const type = typeRaw as ComposerResourceSelection["type"];
    const key = `${type}:${id}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    selections.push({ type, id });
  }
  return selections;
}

export async function stopConversationExecution(conversation: Session): Promise<void> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return;
  }

  const active = runtime.executions.find(
    (item) => item.state === "executing" || item.state === "pending" || item.state === "confirming" || item.state === "awaiting_input"
  );
  if (!active) {
    return;
  }

  try {
    await cancelSessionRun(conversation.id, active.id);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function removeQueuedConversationExecution(
  conversation: Session,
  executionID: string
): Promise<void> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return;
  }

  const normalizedExecutionID = executionID.trim();
  if (normalizedExecutionID === "") {
    return;
  }

  const queuedExecution = runtime.executions.find(
    (execution) => execution.id === normalizedExecutionID && execution.state === "queued"
  );
  if (!queuedExecution) {
    return;
  }

  try {
    await controlRun(queuedExecution.id, "stop");
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function approveConversationExecution(conversation: Session): Promise<void> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return;
  }

  const confirming = runtime.executions.find((item) => item.state === "confirming");
  if (!confirming) {
    return;
  }

  try {
    await controlRun(confirming.id, "approve");
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function denyConversationExecution(conversation: Session): Promise<void> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return;
  }

  const confirming = runtime.executions.find((item) => item.state === "confirming");
  if (!confirming) {
    return;
  }

  try {
    await controlRun(confirming.id, "deny");
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function answerConversationExecutionQuestion(
  conversation: Session,
  input: {
    executionId: string;
    questionId: string;
    selectedOptionId?: string;
    text?: string;
  }
): Promise<void> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return;
  }
  const executionID = input.executionId.trim();
  const questionID = input.questionId.trim();
  const selectedOptionID = input.selectedOptionId?.trim() ?? "";
  const text = input.text?.trim() ?? "";
  if (executionID === "" || questionID === "") {
    return;
  }
  if (selectedOptionID === "" && text === "") {
    return;
  }
  const awaitingInput = runtime.executions.find((item) => item.id === executionID && item.state === "awaiting_input");
  if (!awaitingInput) {
    return;
  }

  try {
    await controlRun(awaitingInput.id, "answer", {
      question_id: questionID,
      selected_option_id: selectedOptionID || undefined,
      text: text || undefined
    });
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function rollbackConversationToMessage(conversationId: string, messageId: string): Promise<void> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return;
  }

  const targetMessage = runtime.messages.find((message) => message.id === messageId);
  if (targetMessage && targetMessage.role !== "user") {
    return;
  }
  const targetQueueIndex = targetMessage?.queue_index ?? 0;

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetQueueIndex, "thinking_delta", {
      stage: "rollback_requested",
      message_id: messageId
    })
  );

  try {
    await rollbackSessionToMessage(conversationId, messageId);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    return;
  }

  try {
    const detail = await getSessionDetail(conversationId);
    hydrateSessionRuntime(
      detail.session,
      runtime.projectKind === "git",
      detail
    );
    setSessionChangeSet(conversationId, null);
    void refreshConversationChangeSet(conversationId);
    return;
  } catch {
    // Fall back to local snapshot recovery if detail refresh fails.
  }

  const snapshot = findSessionSnapshotForMessage(conversationId, messageId);
  if (!snapshot) {
    sessionStore.error = "ROLLBACK_SYNC_FAILED: rollback succeeded but local state refresh failed";
    return;
  }

  runtime.messages = snapshot.messages.map((message) => ({ ...message }));
  runtime.executions = restoreExecutionsFromSnapshot(runtime, conversationId, snapshot);
  runtime.runs = runtime.executions;
  runtime.snapshots = runtime.snapshots.filter((item) => item.created_at <= snapshot.created_at);
  runtime.worktreeRef = snapshot.worktree_ref;
  runtime.inspectorTab = snapshot.inspector_state.tab;
  setSessionChangeSet(conversationId, null);

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetQueueIndex, "thinking_delta", {
      stage: "snapshot_applied",
      message_id: messageId
    })
  );

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetQueueIndex, "thinking_delta", {
      stage: "rollback_completed",
      message_id: messageId
    })
  );

  void refreshConversationChangeSet(conversationId);
}

export async function commitConversationChangeset(conversationId: string, message = ""): Promise<void> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return;
  }
  const current = runtime.changeSet;
  if (!current) {
    sessionStore.error = "CHANGESET_NOT_FOUND: no changeset found for current conversation";
    return;
  }
  if (!current.capability.can_commit) {
    sessionStore.error = current.capability.reason || "CHANGESET_COMMIT_DISABLED: changeset cannot be committed currently";
    return;
  }
  const changeSetID = current.change_set_id.trim();
  if (changeSetID === "") {
    sessionStore.error = "CHANGESET_ID_MISSING: changeset id is required";
    return;
  }
  const finalMessage = message.trim() || current.suggested_message.message.trim();
  if (finalMessage === "") {
    sessionStore.error = "CHANGESET_MESSAGE_REQUIRED: commit message is required";
    return;
  }

  try {
    await commitSessionChangeSet(conversationId, {
      message: finalMessage,
      expected_change_set_id: changeSetID
    });
    await refreshConversationChangeSet(conversationId);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function discardConversationChangeset(conversationId: string): Promise<void> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return;
  }
  const current = runtime.changeSet;
  if (!current) {
    sessionStore.error = "CHANGESET_NOT_FOUND: no changeset found for current conversation";
    return;
  }
  if (!current.capability.can_discard) {
    sessionStore.error = current.capability.reason || "CHANGESET_DISCARD_DISABLED: changeset cannot be discarded currently";
    return;
  }
  const changeSetID = current.change_set_id.trim();
  if (changeSetID === "") {
    sessionStore.error = "CHANGESET_ID_MISSING: changeset id is required";
    return;
  }

  try {
    await discardSessionChangeSet(conversationId, {
      expected_change_set_id: changeSetID
    });
    await refreshConversationChangeSet(conversationId);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function refreshConversationChangeSet(conversationId: string): Promise<void> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return;
  }
  try {
    const changeSet = await getSessionChangeSet(conversationId);
    setSessionChangeSet(conversationId, changeSet);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
  }
}

export async function loadConversationRunTaskGraph(conversationId: string): Promise<ConversationRunTaskGraph | null> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return null;
  }
  const runID = resolveRunContextExecutionID(runtime);
  if (runID === "") {
    return null;
  }
  try {
    return await getRunTaskGraph(runID);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    return null;
  }
}

export async function loadConversationRunTasks(
  conversationId: string,
  options: {
    state?: ConversationRunTaskState;
    cursor?: string;
    limit?: number;
  } = {}
): Promise<ConversationRunTaskListResponse | null> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return null;
  }
  const runID = resolveRunContextExecutionID(runtime);
  if (runID === "") {
    return null;
  }
  try {
    return await listRunTasks(runID, options);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    return null;
  }
}

export async function loadConversationRunTaskById(conversationId: string, taskId: string): Promise<ConversationRunTaskNode | null> {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return null;
  }
  const normalizedTaskID = taskId.trim();
  if (normalizedTaskID === "") {
    return null;
  }
  const runID = resolveRunContextExecutionID(runtime);
  if (runID === "") {
    return null;
  }
  try {
    return await getRunTaskById(runID, normalizedTaskID);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    return null;
  }
}

export async function controlConversationRunTask(
  conversation: Session,
  taskId: string,
  action: ConversationRunTaskControlAction,
  reason?: string
): Promise<ConversationRunTaskControlResponse | null> {
  const runtime = sessionStore.bySessionId[conversation.id];
  if (!runtime) {
    return null;
  }
  const normalizedTaskID = taskId.trim();
  if (normalizedTaskID === "") {
    return null;
  }
  const runID = resolveRunContextExecutionID(runtime);
  if (runID === "") {
    return null;
  }
  try {
    return await controlRunTask(runID, normalizedTaskID, action, reason);
  } catch (error) {
    sessionStore.error = toDisplayError(error);
    return null;
  }
}

export function applyIncomingExecutionEvent(conversationId: string, event: RunLifecycleEvent): void {
  const runtime = sessionStore.bySessionId[conversationId];
  if (!runtime) {
    return;
  }

  const eventDedupKey = buildEventDedupKey(event);
  if (eventDedupKey !== "" && runtime.processedEventKeySet.has(eventDedupKey)) {
    return;
  }
  rememberProcessedEvent(runtime, eventDedupKey);

  appendRuntimeEvent(runtime, event);

  const transition = updateExecutionTransition(runtime, conversationId, event);
  applyDiffUpdate(runtime, event);
  if (
    event.type === "diff_generated" ||
    event.type === "change_set_updated" ||
    event.type === "change_set_committed" ||
    event.type === "change_set_discarded" ||
    event.type === "change_set_rolled_back"
  ) {
    void refreshConversationChangeSet(conversationId);
  }
  appendTerminalMessageFromEvent(runtime, conversationId, event, transition);
}

function isCatalogStaleError(error: unknown): boolean {
  return error instanceof ApiError && error.code === "CATALOG_STALE";
}

function resolveRunContextExecutionID(runtime: { executions: Array<{ id: string; state: string; queue_index: number; created_at: string }> }): string {
  const active = runtime.executions.find((execution) =>
    execution.state === "executing" ||
    execution.state === "pending" ||
    execution.state === "confirming" ||
    execution.state === "awaiting_input"
  );
  if (active?.id.trim()) {
    return active.id.trim();
  }

  const ordered = [...runtime.executions].sort((left, right) => {
    if (left.queue_index !== right.queue_index) {
      return left.queue_index - right.queue_index;
    }
    if (left.created_at !== right.created_at) {
      return left.created_at.localeCompare(right.created_at);
    }
    return left.id.localeCompare(right.id);
  });
  const seed = ordered[0];
  return seed?.id.trim() ?? "";
}
