import { streamConversationEvents } from "@/modules/conversation/services";
import { conversationStore } from "@/modules/conversation/store/state";
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
    onEvent: (event) => {
      const current = conversationStore.byConversationId[conversation.id];
      if (!current) {
        return;
      }
      current.events.push(event as ExecutionEvent);
    },
    onStatusChange: (status) => {
      const current = conversationStore.byConversationId[conversation.id];
      if (!current) {
        return;
      }

      current.status = status;
      if (status !== "connected") {
        current.events.push(
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
  conversationStore.streams[conversationId]?.close();
  delete conversationStore.streams[conversationId];
}
