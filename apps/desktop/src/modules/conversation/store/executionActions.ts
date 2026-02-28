import {
  cancelExecution,
  commitExecution,
  controlExecutionRun,
  discardExecution,
  getComposerCatalog,
  getConversationDetail,
  loadExecutionDiff,
  rollbackExecution,
  submitComposerInput
} from "@/modules/conversation/services";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import {
  dedupeExecutions,
  restoreExecutionsFromSnapshot,
  upsertExecutionFromServer
} from "@/modules/conversation/store/executionRuntime";
import { buildEventDedupKey, rememberProcessedEvent } from "@/modules/conversation/store/executionEventIdempotency";
import {
  appendRuntimeEvent,
  conversationStore,
  createConversationSnapshot,
  ensureConversationRuntime,
  findSnapshotForMessage,
  hydrateConversationRuntime,
  pushConversationSnapshot
} from "@/modules/conversation/store/state";
import {
  applyDiffUpdate,
  appendTerminalMessageFromEvent,
  type ExecutionTransition,
  updateExecutionTransition
} from "@/modules/conversation/store/executionEventHandlers";
import { toDisplayError } from "@/shared/services/errorMapper";
import { ApiError } from "@/shared/services/http";
import { createMockId } from "@/shared/utils/id";
import type { ComposerResourceSelection, Conversation, ConversationMessage, Execution, ExecutionEvent } from "@/shared/types/api";

export async function submitConversationMessage(
  conversation: Conversation,
  isGitProject: boolean,
  options: {
    catalogRevision?: string;
  } = {}
): Promise<void> {
  const runtime = ensureConversationRuntime(conversation, isGitProject);
  const content = runtime.draft.trim();
  if (content === "") {
    return;
  }
  if (runtime.modelId.trim() === "") {
    conversationStore.error = "当前项目未绑定可用模型，请先在项目配置中绑定模型";
    return;
  }

  runtime.draft = "";
  const queueIndex = runtime.executions.length;

  const userMessage: ConversationMessage = {
    id: createMockId("msg"),
    conversation_id: conversation.id,
    role: "user",
    content,
    queue_index: queueIndex,
    can_rollback: true,
    created_at: new Date().toISOString()
  };
  runtime.messages.push(userMessage);
  pushConversationSnapshot(
    conversation.id,
    createConversationSnapshot(runtime, conversation.id, userMessage.id)
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
      response = await submitComposerInput(conversation, baseInput);
    } catch (error) {
      if (isCatalogStaleError(error)) {
        const catalog = await getComposerCatalog(conversation.id);
        response = await submitComposerInput(conversation, {
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
        conversation_id: conversation.id,
        role: "system",
        content: response.command_result.output,
        created_at: new Date().toISOString()
      });
      return;
    }

    upsertExecutionFromServer(runtime, response.execution);
    dedupeExecutions(runtime);
    appendRuntimeEvent(
      runtime,
      createExecutionEvent(conversation.id, response.execution.id, response.queue_index, "message_received", {
        message_id: response.execution.message_id,
        queue_state: response.queue_state
      })
    );
  } catch (error) {
    conversationStore.error = toDisplayError(error);
    runtime.messages.push({
      id: createMockId("msg"),
      conversation_id: conversation.id,
      role: "system",
      content: conversationStore.error,
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

export async function stopConversationExecution(conversation: Conversation): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
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
    await cancelExecution(conversation.id, active.id);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function removeQueuedConversationExecution(
  conversation: Conversation,
  executionID: string
): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
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
    await controlExecutionRun(queuedExecution.id, "stop");
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function approveConversationExecution(conversation: Conversation): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
  if (!runtime) {
    return;
  }

  const confirming = runtime.executions.find((item) => item.state === "confirming");
  if (!confirming) {
    return;
  }

  try {
    await controlExecutionRun(confirming.id, "approve");
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function denyConversationExecution(conversation: Conversation): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
  if (!runtime) {
    return;
  }

  const confirming = runtime.executions.find((item) => item.state === "confirming");
  if (!confirming) {
    return;
  }

  try {
    await controlExecutionRun(confirming.id, "deny");
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function answerConversationExecutionQuestion(
  conversation: Conversation,
  input: {
    executionId: string;
    questionId: string;
    selectedOptionId?: string;
    text?: string;
  }
): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
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
    await controlExecutionRun(awaitingInput.id, "answer", {
      question_id: questionID,
      selected_option_id: selectedOptionID || undefined,
      text: text || undefined
    });
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function rollbackConversationToMessage(conversationId: string, messageId: string): Promise<void> {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  const targetMessage = runtime.messages.find((message) => message.id === messageId);
  if (!targetMessage || targetMessage.role !== "user") {
    return;
  }

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "rollback_requested",
      message_id: messageId
    })
  );

  try {
    await rollbackExecution(conversationId, messageId);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
    return;
  }

  try {
    const detail = await getConversationDetail(conversationId);
    hydrateConversationRuntime(
      detail.conversation,
      runtime.diffCapability.can_commit || runtime.diffCapability.can_discard,
      detail
    );
    runtime.diff = [];
    runtime.diffExecutionId = "";
    return;
  } catch {
    // Fall back to local snapshot recovery if detail refresh fails.
  }

  const snapshot = findSnapshotForMessage(conversationId, messageId);
  if (!snapshot) {
    conversationStore.error = "ROLLBACK_SYNC_FAILED: rollback succeeded but local state refresh failed";
    return;
  }

  runtime.messages = snapshot.messages.map((message) => ({ ...message }));
  runtime.executions = restoreExecutionsFromSnapshot(runtime, conversationId, snapshot);
  runtime.snapshots = runtime.snapshots.filter((item) => item.created_at <= snapshot.created_at);
  runtime.worktreeRef = snapshot.worktree_ref;
  runtime.inspectorTab = snapshot.inspector_state.tab;
  runtime.diff = [];
  runtime.diffExecutionId = "";

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "snapshot_applied",
      message_id: messageId
    })
  );

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "rollback_completed",
      message_id: messageId
    })
  );
}

export async function commitLatestDiff(conversationId: string): Promise<void> {
  const execution = resolveDiffTargetExecution(conversationId);
  if (!execution) {
    conversationStore.error = "DIFF_NOT_FOUND: no execution found for current diff list";
    return;
  }

  try {
    await commitExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
      runtime.diffExecutionId = "";
    }
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function discardLatestDiff(conversationId: string): Promise<void> {
  const execution = resolveDiffTargetExecution(conversationId);
  if (!execution) {
    conversationStore.error = "DIFF_NOT_FOUND: no execution found for current diff list";
    return;
  }

  try {
    await discardExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
      runtime.diffExecutionId = "";
    }
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function refreshExecutionDiff(conversationId: string, executionId: string): Promise<void> {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }
  try {
    runtime.diff = await loadExecutionDiff(executionId);
    runtime.diffExecutionId = executionId;
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

function resolveDiffTargetExecution(conversationId: string): Execution | undefined {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return undefined;
  }
  const targetExecutionId = runtime.diffExecutionId.trim();
  if (targetExecutionId !== "") {
    const matched = runtime.executions.find((execution) => execution.id === targetExecutionId);
    if (matched) {
      return matched;
    }
  }
  return [...runtime.executions]
    .reverse()
    .find((execution) => execution.state === "completed" || execution.state === "failed" || execution.state === "cancelled")
    ?? runtime.executions[runtime.executions.length - 1];
}

export function applyIncomingExecutionEvent(conversationId: string, event: ExecutionEvent): void {
  const runtime = conversationStore.byConversationId[conversationId];
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
    event.execution_id.trim() !== "" &&
    (event.type === "diff_generated" ||
      event.type === "execution_done" ||
      event.type === "execution_error" ||
      event.type === "execution_stopped")
  ) {
    void refreshExecutionDiff(conversationId, event.execution_id);
  }
  appendTerminalMessageFromEvent(runtime, conversationId, event, transition);
}

function isCatalogStaleError(error: unknown): boolean {
  return error instanceof ApiError && error.code === "CATALOG_STALE";
}
