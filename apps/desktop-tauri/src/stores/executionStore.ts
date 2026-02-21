/**
 * executionStore.ts — v0.2.0 执行状态 (替代 runStore.ts)
 *
 * 管理当前会话的 execution 状态、事件流、pending confirmations。
 */
import { create } from "zustand";

import type { EventEnvelope } from "@/types/generated";
import type { ToolCallView } from "@/types/ui";

export interface PendingConfirmation {
  executionId: string;
  callId: string;
  toolName: string;
  args: Record<string, unknown>;
  riskLevel: string;
  createdAt: string;
}

interface ExecutionState {
  executionId: string | null;
  sessionId: string | null;
  traceId: string | null;
  status: "idle" | "executing" | "waiting_confirmation" | "completed" | "failed" | "cancelled";
  events: EventEnvelope[];
  pendingConfirmations: PendingConfirmation[];
  lastPatch: string | null;
  lastPlan: { summary?: string; steps?: string[] } | null;
  toolCalls: Record<string, ToolCallView>;
  selectedToolCallId: string | null;
  lastSeq: number;

  // Actions
  startExecution: (executionId: string, sessionId: string, traceId: string) => void;
  reset: () => void;
  appendRawEvent: (type: string, payloadJson: string, seq: number) => void;
  resolvePendingConfirmation: (callId: string, approved: boolean) => void;
  setSelectedToolCallId: (callId: string | null) => void;
  setStatus: (status: ExecutionState["status"]) => void;
}

const INITIAL_STATE = {
  executionId: null,
  sessionId: null,
  traceId: null,
  status: "idle" as const,
  events: [],
  pendingConfirmations: [],
  lastPatch: null,
  lastPlan: null,
  toolCalls: {},
  selectedToolCallId: null,
  lastSeq: 0,
};

export const useExecutionStore = create<ExecutionState>((set, get) => ({
  ...INITIAL_STATE,

  startExecution: (executionId, sessionId, traceId) =>
    set({
      ...INITIAL_STATE,
      executionId,
      sessionId,
      traceId,
      status: "executing",
    }),

  reset: () => set(INITIAL_STATE),

  setStatus: (status) => set({ status }),

  setSelectedToolCallId: (callId) => set({ selectedToolCallId: callId }),

  appendRawEvent: (type, payloadJson, seq) => {
    let payload: Record<string, unknown> = {};
    try {
      payload = JSON.parse(payloadJson);
    } catch {
      payload = { raw: payloadJson };
    }

    const executionId = get().executionId ?? "";
    const event: EventEnvelope = {
      protocol_version: "2.0.0",
      trace_id: get().traceId ?? "",
      event_id: `${executionId}:${seq}`,
      execution_id: executionId,
      seq,
      ts: new Date().toISOString(),
      type: type as EventEnvelope["type"],
      payload,
    };

    const nextEvents = [...get().events, event].sort((a, b) => a.seq - b.seq);
    const nextPending = [...get().pendingConfirmations];
    const nextToolCalls = { ...get().toolCalls };
    let nextPatch = get().lastPatch;
    let nextPlan = get().lastPlan;
    let nextSelected = get().selectedToolCallId;
    let nextStatus = get().status;

    if (type === "tool_call") {
      const callId = String(payload.call_id ?? "");
      if (callId) {
        nextToolCalls[callId] = {
          callId,
          toolName: String(payload.tool_name ?? "unknown"),
          args: (payload.args ?? {}) as Record<string, unknown>,
          requiresConfirmation: Boolean(payload.requires_confirmation),
          status: payload.requires_confirmation ? "waiting" : "completed",
          createdAt: event.ts,
        };
        if (!nextSelected) nextSelected = callId;
      }
    }

    if (type === "confirmation_request") {
      const callId = String(payload.call_id ?? "");
      if (callId) {
        const existing = nextToolCalls[callId];
        nextPending.push({
          executionId,
          callId,
          toolName: String(payload.tool_name ?? existing?.toolName ?? ""),
          args: (payload.args ?? existing?.args ?? {}) as Record<string, unknown>,
          riskLevel: String(payload.risk_level ?? "medium"),
          createdAt: event.ts,
        });
        nextStatus = "waiting_confirmation";
      }
    }

    if (type === "tool_result") {
      const callId = String(payload.call_id ?? "");
      const cur = nextToolCalls[callId];
      if (cur) {
        nextToolCalls[callId] = {
          ...cur,
          output: payload.ok === true ? payload.output : payload.error,
          finishedAt: event.ts,
          status: payload.ok === true ? "completed" : "failed",
        };
      }
    }

    if (type === "plan") {
      nextPlan = {
        summary: payload.summary as string | undefined,
        steps: Array.isArray(payload.steps) ? (payload.steps as string[]) : undefined,
      };
    }

    if (type === "patch") {
      nextPatch = String(payload.unified_diff ?? "");
    }

    if (type === "done") {
      const doneStatus = String(payload.status ?? "completed");
      nextStatus = doneStatus === "completed" ? "completed" : "failed";
    }

    if (type === "cancelled") {
      nextStatus = "cancelled";
    }

    if (type === "confirmation_decision") {
      const callId = String(payload.call_id ?? "");
      const approved = payload.decision === "approved";
      const pendingIdx = nextPending.findIndex((pending) => pending.callId === callId);
      if (pendingIdx >= 0) {
        nextPending.splice(pendingIdx, 1);
      }
      if (nextToolCalls[callId]) {
        nextToolCalls[callId] = {
          ...nextToolCalls[callId],
          status: approved ? "approved" : "denied",
        };
      }
      if (nextStatus === "waiting_confirmation") {
        nextStatus = "executing";
      }
    }

    set({
      events: nextEvents,
      pendingConfirmations: nextPending,
      toolCalls: nextToolCalls,
      lastPatch: nextPatch,
      lastPlan: nextPlan,
      selectedToolCallId: nextSelected,
      status: nextStatus,
      lastSeq: Math.max(get().lastSeq, seq),
    });
  },

  resolvePendingConfirmation: (callId, approved) => {
    const cur = get().toolCalls[callId];
    set((state) => ({
      pendingConfirmations: state.pendingConfirmations.filter((p) => p.callId !== callId),
      toolCalls: cur
        ? { ...state.toolCalls, [callId]: { ...cur, status: approved ? "approved" : "denied" } }
        : state.toolCalls,
    }));
  },
}));
