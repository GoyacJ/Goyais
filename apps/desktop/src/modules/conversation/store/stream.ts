import { streamConversationEvents } from "@/modules/conversation/services";
import { applyIncomingExecutionEvent } from "@/modules/conversation/store/executionActions";
import { appendRuntimeEvent, conversationStore } from "@/modules/conversation/store/state";
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

  conversationStore.streams[conversation.id] = streamConversationEvents(conversation.id, {
    token,
    initialLastEventId: runtime.lastEventId,
    onEvent: (event) => {
      const incoming = normalizeExecutionEvent(event, conversation.id);
      if (!incoming) {
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
