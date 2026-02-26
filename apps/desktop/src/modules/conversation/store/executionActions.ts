import {
  cancelExecution,
  commitExecution,
  discardExecution,
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
  getLatestFinishedExecution,
  pushConversationSnapshot
} from "@/modules/conversation/store/state";
import {
  applyDiffUpdate,
  appendTerminalMessageFromEvent,
  type ExecutionTransition,
  updateExecutionTransition
} from "@/modules/conversation/store/executionEventHandlers";
import { toDisplayError } from "@/shared/services/errorMapper";
import { createMockId } from "@/shared/utils/id";
import type { ComposerResourceSelection, Conversation, ConversationMessage, ExecutionEvent } from "@/shared/types/api";

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
    const response = await submitComposerInput(conversation, {
      raw_input: content,
      mode: runtime.mode,
      model_config_id: runtime.modelId.trim() || undefined,
      selected_resources: extractSelectedResources(content),
      catalog_revision: options.catalogRevision
    });

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

  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "pending");
  if (!active) {
    return;
  }

  try {
    await cancelExecution(conversation.id, active.id);
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
  }

  const snapshot = findSnapshotForMessage(conversationId, messageId);
  if (!snapshot) {
    return;
  }

  runtime.messages = snapshot.messages.map((message) => ({ ...message }));
  runtime.executions = restoreExecutionsFromSnapshot(runtime, conversationId, snapshot);
  runtime.snapshots = runtime.snapshots.filter((item) => item.created_at <= snapshot.created_at);
  runtime.worktreeRef = snapshot.worktree_ref;
  runtime.inspectorTab = snapshot.inspector_state.tab;
  runtime.diff = [];

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
  const execution = getLatestFinishedExecution(conversationId);
  if (!execution) {
    return;
  }

  try {
    await commitExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
    }
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function discardLatestDiff(conversationId: string): Promise<void> {
  const execution = getLatestFinishedExecution(conversationId);
  if (!execution) {
    return;
  }

  try {
    await discardExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
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
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
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
  appendTerminalMessageFromEvent(runtime, conversationId, event, transition);
}
