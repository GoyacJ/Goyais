import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  applyIncomingExecutionEvent,
  ensureConversationRuntime,
  getExecutionStateCounts,
  resetConversationStore,
  setConversationDraft,
  submitConversationMessage
} from "@/modules/conversation/store";
import type { Conversation } from "@/shared/types/api";

const mockConversation: Conversation = {
  id: "conv_race",
  workspace_id: "ws_local",
  project_id: "proj_1",
  name: "Race Conversation",
  queue_state: "idle",
  default_mode: "agent",
  model_id: "gpt-5.3",
  base_revision: 0,
  active_execution_id: null,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:00Z"
};

describe("conversation execution race", () => {
  beforeEach(() => {
    resetConversationStore();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps a single terminal execution when done arrives before create response", async () => {
    let resolveCreate: (() => void) | undefined;
    const createResponse = {
      execution: {
        id: "exec_race_1",
        workspace_id: "ws_local",
        conversation_id: mockConversation.id,
        message_id: "msg_race_1",
        state: "pending" as const,
        mode: "agent" as const,
        model_id: "gpt-5.3",
        mode_snapshot: "agent" as const,
        model_snapshot: {
          model_id: "gpt-5.3"
        },
        project_revision_snapshot: 0,
        queue_index: 0,
        trace_id: "tr_race_1",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      },
      queue_state: "running" as const,
      queue_index: 0
    };

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";
      if (url.endsWith(`/v1/conversations/${mockConversation.id}/messages`) && method === "POST") {
        return new Promise<Response>((resolve) => {
          resolveCreate = () => resolve(jsonResponse(createResponse, 201));
        });
      }
      return Promise.resolve(jsonResponse({ code: "ROUTE_NOT_FOUND" }, 404));
    });
    vi.stubGlobal("fetch", fetchMock);

    ensureConversationRuntime(mockConversation, true);
    setConversationDraft(mockConversation.id, "race message");
    const submitPromise = submitConversationMessage(mockConversation, true);

    await Promise.resolve();
    applyIncomingExecutionEvent(mockConversation.id, {
      event_id: "evt_race_done",
      execution_id: "exec_race_1",
      conversation_id: mockConversation.id,
      trace_id: "tr_race_1",
      sequence: 1,
      queue_index: 0,
      type: "execution_done",
      timestamp: new Date().toISOString(),
      payload: {
        content: "done"
      }
    });

    resolveCreate?.();
    await submitPromise;

    const runtime = ensureConversationRuntime(mockConversation, true);
    expect(runtime.executions.length).toBe(1);
    expect(runtime.executions[0]?.id).toBe("exec_race_1");
    expect(runtime.executions[0]?.state).toBe("completed");

    const counts = getExecutionStateCounts(runtime);
    expect(counts.pending).toBe(0);
    expect(counts.executing).toBe(0);
  });
});

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json"
    }
  });
}
