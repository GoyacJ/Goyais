import { create } from "zustand";

import type { EventEnvelope } from "../types/generated";

interface PendingConfirmation {
  runId: string;
  callId: string;
  toolName: string;
  args: Record<string, unknown>;
}

interface RunState {
  runId?: string;
  events: EventEnvelope[];
  pendingConfirmations: PendingConfirmation[];
  lastPatch?: string;
  setRunId: (runId: string) => void;
  reset: () => void;
  appendEvent: (event: EventEnvelope) => void;
  popPendingConfirmation: () => PendingConfirmation | undefined;
}

export const useRunStore = create<RunState>((set, get) => ({
  events: [],
  pendingConfirmations: [],
  setRunId: (runId) => set({ runId }),
  reset: () => set({ runId: undefined, events: [], pendingConfirmations: [], lastPatch: undefined }),
  appendEvent: (event) => {
    const nextEvents = [...get().events, event].sort((a, b) => a.seq - b.seq);
    const nextPending = [...get().pendingConfirmations];
    let nextPatch = get().lastPatch;

    if (event.type === "tool_call" && event.payload.requires_confirmation === true) {
      nextPending.push({
        runId: event.run_id,
        callId: String(event.payload.call_id),
        toolName: String(event.payload.tool_name),
        args: (event.payload.args ?? {}) as Record<string, unknown>
      });
    }

    if (event.type === "patch") {
      nextPatch = String(event.payload.unified_diff ?? "");
    }

    set({ events: nextEvents, pendingConfirmations: nextPending, lastPatch: nextPatch });
  },
  popPendingConfirmation: () => {
    const [head, ...tail] = get().pendingConfirmations;
    set({ pendingConfirmations: tail });
    return head;
  }
}));
