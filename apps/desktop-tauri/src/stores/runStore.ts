import { create } from "zustand";

import type { EventEnvelope } from "../types/generated";
import type { ToolCallView } from "../types/ui";

export interface PendingConfirmation {
  runId: string;
  callId: string;
  toolName: string;
  args: Record<string, unknown>;
  createdAt: string;
}

interface RunContext {
  projectId: string;
  modelConfigId: string;
  workspacePath: string;
  sessionId: string;
}

interface RunState {
  runId?: string;
  events: EventEnvelope[];
  pendingConfirmations: PendingConfirmation[];
  lastPatch?: string;
  selectedToolCallId?: string;
  toolCalls: Record<string, ToolCallView>;
  context: RunContext;
  setRunId: (runId: string) => void;
  setContext: (context: Partial<RunContext>) => void;
  setSelectedToolCallId: (callId?: string) => void;
  reset: () => void;
  appendEvent: (event: EventEnvelope) => void;
  resolvePendingConfirmation: (callId: string, approved: boolean) => void;
}

export const useRunStore = create<RunState>((set, get) => ({
  events: [],
  pendingConfirmations: [],
  toolCalls: {},
  context: {
    projectId: "project-demo",
    modelConfigId: "model-demo",
    workspacePath: "/Users/goya/Repo/Git/Goyais",
    sessionId: "session-demo"
  },
  setRunId: (runId) => set({ runId }),
  setContext: (context) =>
    set((state) => ({
      context: {
        ...state.context,
        ...context
      }
    })),
  setSelectedToolCallId: (callId) => set({ selectedToolCallId: callId }),
  reset: () =>
    set({
      runId: undefined,
      events: [],
      pendingConfirmations: [],
      lastPatch: undefined,
      selectedToolCallId: undefined,
      toolCalls: {}
    }),
  appendEvent: (event) => {
    const nextEvents = [...get().events, event].sort((a, b) => a.seq - b.seq);
    const nextPending = [...get().pendingConfirmations];
    let nextPatch = get().lastPatch;
    const nextToolCalls = { ...get().toolCalls };
    let nextSelected = get().selectedToolCallId;

    if (event.type === "tool_call" && event.payload.requires_confirmation === true) {
      nextPending.push({
        runId: event.run_id,
        callId: String(event.payload.call_id),
        toolName: String(event.payload.tool_name),
        args: (event.payload.args ?? {}) as Record<string, unknown>,
        createdAt: event.ts
      });
    }

    if (event.type === "tool_call") {
      const callId = String(event.payload.call_id ?? "");
      if (callId) {
        nextToolCalls[callId] = {
          callId,
          toolName: String(event.payload.tool_name ?? "unknown"),
          args: (event.payload.args ?? {}) as Record<string, unknown>,
          requiresConfirmation: Boolean(event.payload.requires_confirmation),
          status: event.payload.requires_confirmation ? "waiting" : "completed",
          createdAt: event.ts
        };
        if (!nextSelected) {
          nextSelected = callId;
        }
      }
    }

    if (event.type === "tool_result") {
      const callId = String(event.payload.call_id ?? "");
      const current = nextToolCalls[callId];
      if (current) {
        nextToolCalls[callId] = {
          ...current,
          output: event.payload.output,
          finishedAt: event.ts,
          status: event.payload.ok === true ? "completed" : "failed"
        };
      }
    }

    if (event.type === "patch") {
      nextPatch = String(event.payload.unified_diff ?? "");
    }

    set({
      events: nextEvents,
      pendingConfirmations: nextPending,
      lastPatch: nextPatch,
      toolCalls: nextToolCalls,
      selectedToolCallId: nextSelected
    });
  },
  resolvePendingConfirmation: (callId, approved) => {
    const current = get().toolCalls[callId];
    set((state) => ({
      pendingConfirmations: state.pendingConfirmations.filter((item) => item.callId !== callId),
      toolCalls: current
        ? {
            ...state.toolCalls,
            [callId]: {
              ...current,
              status: approved ? "approved" : "denied"
            }
          }
        : state.toolCalls
    }));
  }
}));
