import { getConversationDetail, streamConversationEvents } from "@/modules/conversation/services";
import { applyIncomingExecutionEvent } from "@/modules/conversation/store/executionActions";
import {
  appendRuntimeEvent,
  conversationStore,
  hydrateConversationRuntime
} from "@/modules/conversation/store/state";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import type { Conversation, ExecutionEvent } from "@/shared/types/api";

export function attachConversationStream(conversation: Conversation, token?: string): void {
  if (typeof EventSource === "undefined") {
    return;
  }

  const runtime = conversationStore.byConversationId[conversation.id];
  if (!runtime || conversationStore.streams[conversation.id]) {
    return;
  }
  let resyncInFlight = false;

  conversationStore.streams[conversation.id] = streamConversationEvents(conversation.id, {
    token,
    initialLastEventId: runtime.lastEventId,
    onEvent: (event) => {
      const incoming = normalizeExecutionEvent(event, conversation.id);
      if (!incoming) {
        return;
      }
      if (isSSEBackfillResyncEvent(incoming)) {
        const latestEventID = resolveLatestEventIDFromResyncPayload(incoming);
        runtime.lastEventId = latestEventID;
        if (resyncInFlight) {
          return;
        }
        resyncInFlight = true;
        void getConversationDetail(conversation.id, { token })
          .then((detail) => {
            const current = conversationStore.byConversationId[conversation.id];
            if (!current) {
              return;
            }
            const isGitProject = current.diffCapability.can_commit;
            hydrateConversationRuntime(conversation, isGitProject, detail);
            if (latestEventID !== "") {
              current.lastEventId = latestEventID;
            }
          })
          .catch((error) => {
            conversationStore.error = toError(error).message;
          })
          .finally(() => {
            resyncInFlight = false;
          });
        return;
      }
      if (resyncInFlight) {
        return;
      }
      const incomingEventID = incoming.event_id?.trim();
      if (incomingEventID) {
        runtime.lastEventId = incomingEventID;
      }
      const eventConversationId = incoming.conversation_id.trim();
      if (eventConversationId !== conversation.id) {
        console.warn(
          `[conversation-stream] routed event by event.conversation_id, stream=${conversation.id}, event=${eventConversationId}`
        );
      }
      const current = conversationStore.byConversationId[eventConversationId];
      if (!current) {
        return;
      }
      applyIncomingExecutionEvent(eventConversationId, incoming);
    },
    onStatusChange: (status) => {
      const current = conversationStore.byConversationId[conversation.id];
      if (!current) {
        return;
      }

      current.status = status;
      if (status !== "connected") {
        appendRuntimeEvent(
          current,
          createExecutionEvent(conversation.id, "", 0, "thinking_delta", {
            sse_status: status
          })
        );
      }
    },
    onError: (error) => {
      conversationStore.error = error.message;
    }
  });
}

export function detachConversationStream(conversationId: string): void {
  const handle = conversationStore.streams[conversationId];
  const runtime = conversationStore.byConversationId[conversationId];
  if (handle && runtime) {
    const lastEventID = handle.lastEventId().trim();
    if (lastEventID !== "") {
      runtime.lastEventId = lastEventID;
    }
  }
  handle?.close();
  delete conversationStore.streams[conversationId];
}

function isSSEBackfillResyncEvent(event: ExecutionEvent): boolean {
  if (event.type !== "thinking_delta") {
    return false;
  }
  const payload = event.payload;
  if (!payload || typeof payload !== "object") {
    return false;
  }
  return payload.resync_required === true && payload.reason === "last_event_id_not_found";
}

function resolveLatestEventIDFromResyncPayload(event: ExecutionEvent): string {
  const raw = event.payload?.latest_event_id;
  if (typeof raw !== "string") {
    return "";
  }
  return raw.trim();
}

function normalizeExecutionEvent(raw: unknown, fallbackConversationId: string): ExecutionEvent | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }

  const candidate = raw as Partial<ExecutionEvent>;
  if (typeof candidate.type !== "string") {
    return null;
  }
  const normalizedConversationId = typeof candidate.conversation_id === "string" && candidate.conversation_id.trim() !== ""
    ? candidate.conversation_id.trim()
    : fallbackConversationId;

  return {
    ...candidate,
    conversation_id: normalizedConversationId
  } as ExecutionEvent;
}

function toError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }
  return new Error("Unknown conversation stream error");
}
