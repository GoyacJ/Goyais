import { getSessionDetail, streamSessionEvents } from "@/modules/conversation/services";
import { applyIncomingExecutionEvent, refreshConversationChangeSet } from "@/modules/conversation/store/executionActions";
import {
  appendRuntimeEvent,
  conversationStore,
  hydrateConversationRuntime
} from "@/modules/conversation/store/state";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import type { ExecutionEvent, RunEventType, Session, StreamRunEventType } from "@/shared/types/api";

const runEventTypes: readonly StreamRunEventType[] = [
  "run_queued",
  "run_started",
  "run_output_delta",
  "run_approval_needed",
  "run_completed",
  "run_failed",
  "run_cancelled"
];

const runToExecutionTypeMap: Record<Exclude<StreamRunEventType, "run_output_delta" | "run_approval_needed">, RunEventType> = {
  run_queued: "message_received",
  run_started: "execution_started",
  run_completed: "execution_done",
  run_failed: "execution_error",
  run_cancelled: "execution_stopped"
};

export function attachConversationStream(session: Session, token?: string): void {
  if (typeof EventSource === "undefined") {
    return;
  }

  const runtime = conversationStore.byConversationId[session.id];
  if (!runtime || conversationStore.streams[session.id]) {
    return;
  }
  let resyncInFlight = false;

  conversationStore.streams[session.id] = streamSessionEvents(session.id, {
    token,
    initialLastEventId: runtime.lastEventId,
    onEvent: (event) => {
      const incoming = normalizeExecutionEvent(event, session.id);
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
        void getSessionDetail(session.id, { token })
          .then((detail) => {
            const current = conversationStore.byConversationId[session.id];
            if (!current) {
              return;
            }
            const isGitProject = current.projectKind === "git";
            hydrateConversationRuntime(session, isGitProject, detail);
            if (latestEventID !== "") {
              current.lastEventId = latestEventID;
            }
            void refreshConversationChangeSet(session.id);
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
      if (eventConversationId !== session.id) {
        console.warn(
          `[session-stream] routed event by event.conversation_id, stream=${session.id}, event=${eventConversationId}`
        );
      }
      const current = conversationStore.byConversationId[eventConversationId];
      if (!current) {
        return;
      }
      applyIncomingExecutionEvent(eventConversationId, incoming);
    },
    onStatusChange: (status) => {
      const current = conversationStore.byConversationId[session.id];
      if (!current) {
        return;
      }

      current.status = status;
      if (status !== "connected") {
        appendRuntimeEvent(
          current,
          createExecutionEvent(session.id, "", 0, "thinking_delta", {
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
  if (!isRecord(raw)) {
    return null;
  }
  const rawType = asString(raw.type);
  if (rawType === "" || !isRunEventType(rawType)) {
    return null;
  }
  return mapRunEventToExecutionEvent(raw, rawType, fallbackConversationId);
}

function toError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }
  return new Error("Unknown conversation stream error");
}

function mapRunEventToExecutionEvent(
  raw: Record<string, unknown>,
  runType: StreamRunEventType,
  fallbackConversationId: string
): ExecutionEvent {
  const payload = asRecord(raw.payload);
  const queueIndex = asInteger(raw.queue_index, asInteger(payload.queue_index, 0));
  const traceId = asString(raw.trace_id) || asString(payload.trace_id);
  const conversationId = resolveConversationID(asString(raw.session_id), fallbackConversationId);

  if (runType === "run_output_delta") {
    return {
      event_id: asString(raw.event_id),
      execution_id: asString(raw.run_id),
      conversation_id: conversationId,
      trace_id: traceId,
      sequence: asInteger(raw.sequence, 0),
      queue_index: queueIndex,
      type: resolveExecutionTypeForRunOutputDelta(payload),
      timestamp: asTimestamp(raw.timestamp),
      payload
    };
  }

  if (runType === "run_approval_needed") {
    return {
      event_id: asString(raw.event_id),
      execution_id: asString(raw.run_id),
      conversation_id: conversationId,
      trace_id: traceId,
      sequence: asInteger(raw.sequence, 0),
      queue_index: queueIndex,
      type: "thinking_delta",
      timestamp: asTimestamp(raw.timestamp),
      payload: {
        ...payload,
        stage: asString(payload.stage) || "run_approval_needed",
        run_state: asString(payload.run_state) || "waiting_approval"
      }
    };
  }

  return {
    event_id: asString(raw.event_id),
    execution_id: asString(raw.run_id),
    conversation_id: conversationId,
    trace_id: traceId,
    sequence: asInteger(raw.sequence, 0),
    queue_index: queueIndex,
    type: runToExecutionTypeMap[runType],
    timestamp: asTimestamp(raw.timestamp),
    payload
  };
}

function resolveExecutionTypeForRunOutputDelta(payload: Record<string, unknown>): RunEventType {
  const explicitEventType = asString(payload.event_type);
  if (
    explicitEventType === "change_set_updated" ||
    explicitEventType === "change_set_committed" ||
    explicitEventType === "change_set_discarded" ||
    explicitEventType === "change_set_rolled_back"
  ) {
    return explicitEventType;
  }
  if (Array.isArray(payload.diff)) {
    return "diff_generated";
  }
  if (asString(payload.call_id) !== "") {
    if (payload.output !== undefined || typeof payload.ok === "boolean") {
      return "tool_result";
    }
    return "tool_call";
  }

  const hasToolName = asString(payload.name) !== "";
  if (hasToolName && payload.output !== undefined) {
    return "tool_result";
  }
  if (hasToolName && payload.input !== undefined) {
    return "tool_call";
  }
  return "thinking_delta";
}

function resolveConversationID(rawConversationID: string, fallbackConversationID: string): string {
  const trimmed = rawConversationID.trim();
  if (trimmed !== "") {
    return trimmed;
  }
  return fallbackConversationID.trim();
}

function isRunEventType(value: string): value is StreamRunEventType {
  return runEventTypes.includes(value as StreamRunEventType);
}

function asString(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

function asInteger(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number.parseInt(value, 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return fallback;
}

function asTimestamp(value: unknown): string {
  const trimmed = asString(value);
  if (trimmed !== "") {
    return trimmed;
  }
  return new Date().toISOString();
}

function asRecord(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) {
    return {};
  }
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
