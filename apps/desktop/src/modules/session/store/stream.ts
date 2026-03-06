import { getSessionDetail, streamSessionEvents } from "@/modules/session/services";
import { applyIncomingExecutionEvent, refreshConversationChangeSet } from "@/modules/session/store/executionActions";
import {
  appendRuntimeEvent,
  hydrateSessionRuntime,
  sessionStore
} from "@/modules/session/store/state";
import { createExecutionEvent } from "@/modules/session/store/events";
import type { RunEventType, RunLifecycleEvent, Session, StreamRunEventType } from "@/shared/types/api";

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

export function attachSessionStream(session: Session, token?: string): void {
  if (typeof EventSource === "undefined") {
    return;
  }

  const runtime = sessionStore.bySessionId[session.id];
  if (!runtime || sessionStore.sessionStreams[session.id]) {
    return;
  }
  let resyncInFlight = false;

  sessionStore.sessionStreams[session.id] = streamSessionEvents(session.id, {
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
            const current = sessionStore.bySessionId[session.id];
            if (!current) {
              return;
            }
            const isGitProject = current.projectKind === "git";
            hydrateSessionRuntime(session, isGitProject, detail);
            if (latestEventID !== "") {
              current.lastEventId = latestEventID;
            }
            void refreshConversationChangeSet(session.id);
          })
          .catch((error) => {
            sessionStore.error = toError(error).message;
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
      const eventSessionId = incoming.session_id.trim();
      if (eventSessionId !== session.id) {
        console.warn(
          `[session-stream] routed event by event.session_id, stream=${session.id}, event=${eventSessionId}`
        );
      }
      const current = sessionStore.bySessionId[eventSessionId];
      if (!current) {
        return;
      }
      applyIncomingExecutionEvent(eventSessionId, incoming);
    },
    onStatusChange: (status) => {
      const current = sessionStore.bySessionId[session.id];
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
      sessionStore.error = error.message;
    }
  });
}

export function attachConversationStream(session: Session, token?: string): void {
  attachSessionStream(session, token);
}

export function detachSessionStream(sessionId: string): void {
  const handle = sessionStore.sessionStreams[sessionId];
  const runtime = sessionStore.bySessionId[sessionId];
  if (handle && runtime) {
    const lastEventID = handle.lastEventId().trim();
    if (lastEventID !== "") {
      runtime.lastEventId = lastEventID;
    }
  }
  handle?.close();
  delete sessionStore.sessionStreams[sessionId];
}

export function detachConversationStream(conversationId: string): void {
  detachSessionStream(conversationId);
}

function isSSEBackfillResyncEvent(event: RunLifecycleEvent): boolean {
  if (event.type !== "thinking_delta") {
    return false;
  }
  const payload = event.payload;
  if (!payload || typeof payload !== "object") {
    return false;
  }
  return payload.resync_required === true && payload.reason === "last_event_id_not_found";
}

function resolveLatestEventIDFromResyncPayload(event: RunLifecycleEvent): string {
  const raw = event.payload?.latest_event_id;
  if (typeof raw !== "string") {
    return "";
  }
  return raw.trim();
}

function normalizeExecutionEvent(raw: unknown, sessionIdHint: string): RunLifecycleEvent | null {
  if (!isRecord(raw)) {
    return null;
  }
  const rawType = asString(raw.type);
  if (rawType === "" || !isRunEventType(rawType)) {
    return null;
  }
  return mapRunEventToExecutionEvent(raw, rawType, sessionIdHint);
}

function toError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }
  return new Error("Unknown session stream error");
}

function mapRunEventToExecutionEvent(
  raw: Record<string, unknown>,
  runType: StreamRunEventType,
  sessionIdHint: string
): RunLifecycleEvent {
  const payload = asRecord(raw.payload);
  const queueIndex = asInteger(raw.queue_index, asInteger(payload.queue_index, 0));
  const traceId = asString(raw.trace_id) || asString(payload.trace_id);
  const conversationId = resolveSessionID(asString(raw.session_id), sessionIdHint);

  if (runType === "run_output_delta") {
    return {
      event_id: asString(raw.event_id),
      run_id: asString(raw.run_id),
      session_id: conversationId,
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
      run_id: asString(raw.run_id),
      session_id: conversationId,
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
    run_id: asString(raw.run_id),
    session_id: conversationId,
    trace_id: traceId,
    sequence: asInteger(raw.sequence, 0),
    queue_index: queueIndex,
    type: runToExecutionTypeMap[runType],
    timestamp: asTimestamp(raw.timestamp),
    payload
  };
}

function resolveExecutionTypeForRunOutputDelta(payload: Record<string, unknown>): RunEventType {
  const source = asString(payload.source).toLowerCase();
  const hookEvent = asString(payload.event).toLowerCase();
  if (
    source === "hook_policy" ||
    hookEvent === "user_prompt_submit" ||
    hookEvent === "pre_tool_use" ||
    hookEvent === "permission_request" ||
    hookEvent === "post_tool_use" ||
    hookEvent === "post_tool_use_failure"
  ) {
    return "thinking_delta";
  }

  const stage = asString(payload.stage).toLowerCase();
  if (stage === "tool_call") {
    return "tool_call";
  }
  if (stage === "tool_result") {
    return "tool_result";
  }
  if (
    stage === "run_approval_needed" ||
    stage === "run_user_question_needed" ||
    stage === "run_user_question_resolved" ||
    stage === "approval_resolved"
  ) {
    return "thinking_delta";
  }

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
  const callID = asString(payload.call_id);
  const hasToolName = asString(payload.name) !== "";
  const hasToolInput = payload.input !== undefined;
  if (callID !== "" && (hasToolName || hasToolInput)) {
    if (payload.output !== undefined || typeof payload.ok === "boolean") {
      return "tool_result";
    }
    return "tool_call";
  }

  if (hasToolName && payload.output !== undefined) {
    return "tool_result";
  }
  if (hasToolName && payload.input !== undefined) {
    return "tool_call";
  }
  return "thinking_delta";
}

function resolveSessionID(rawSessionID: string, sessionIdHint: string): string {
  const trimmed = rawSessionID.trim();
  if (trimmed !== "") {
    return trimmed;
  }
  return sessionIdHint.trim();
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

function asInteger(value: unknown, defaultValue: number): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number.parseInt(value, 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return defaultValue;
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
