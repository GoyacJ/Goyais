import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  applyIncomingExecutionEvent,
  ensureConversationRuntime,
  rollbackConversationToMessage,
  resetConversationStore,
  setConversationDraft,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import type { Conversation } from "@/shared/types/api";

const mockConversation: Conversation = {
  id: "conv_test",
  workspace_id: "ws_local",
  project_id: "proj_1",
  name: "Test Conversation",
  queue_state: "idle",
  default_mode: "agent",
  model_id: "gpt-5.3",
  base_revision: 0,
  active_execution_id: null,
  created_at: "2026-02-23T00:00:00Z",
  updated_at: "2026-02-23T00:00:00Z"
};

describe("conversation store", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    resetConversationStore();
    let executionCounter = 0;
    fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";

      if (url.includes("/v1/conversations/") && url.endsWith("/messages") && method === "POST") {
        executionCounter += 1;
        return jsonResponse(
          {
            execution: {
              id: `exec_${executionCounter}`,
              workspace_id: "ws_local",
              conversation_id: mockConversation.id,
              message_id: `msg_${executionCounter}`,
              state: executionCounter === 1 ? "pending" : "queued",
              mode: "agent",
              model_id: "gpt-5.3",
              mode_snapshot: "agent",
              model_snapshot: {
                model_id: "gpt-5.3"
              },
              project_revision_snapshot: 0,
              queue_index: executionCounter - 1,
              trace_id: `tr_exec_${executionCounter}`,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString()
            },
            queue_state: executionCounter === 1 ? "running" : "queued",
            queue_index: executionCounter - 1
          },
          201
        );
      }

      if (url.endsWith("/stop") && method === "POST") {
        return jsonResponse({ ok: true });
      }

      if (url.includes("/rollback") && method === "POST") {
        return jsonResponse({ ok: true });
      }

      if (url.includes("/v1/executions/") && url.endsWith("/diff") && method === "GET") {
        return jsonResponse([
          {
            id: "diff_1",
            path: "src/main.ts",
            change_type: "modified",
            summary: "queue updated"
          }
        ]);
      }

      return jsonResponse(
        {
          code: "ROUTE_NOT_FOUND",
          message: "Not found",
          details: {},
          trace_id: "tr_not_found"
        },
        404
      );
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("submits messages and keeps server-driven execution states", async () => {
    ensureConversationRuntime(mockConversation, true);
    setConversationDraft(mockConversation.id, "first message");
    await submitConversationMessage(mockConversation, true);

    setConversationDraft(mockConversation.id, "second message");
    await submitConversationMessage(mockConversation, true);

    const runtime = ensureConversationRuntime(mockConversation, true);
    expect(runtime.executions.length).toBe(2);
    expect(runtime.executions[0]?.state).toBe("pending");
    expect(runtime.executions[1]?.state).toBe("queued");
  });

  it("applies incoming execution events to runtime", () => {
    const runtime = ensureConversationRuntime(mockConversation, true);
    runtime.executions.push({
      id: "exec_1",
      workspace_id: "ws_local",
      conversation_id: mockConversation.id,
      message_id: "msg_1",
      state: "pending",
      mode: "agent",
      model_id: "gpt-5.3",
      mode_snapshot: "agent",
      model_snapshot: {
        model_id: "gpt-5.3"
      },
      project_revision_snapshot: 0,
      queue_index: 0,
      trace_id: "tr_exec_1",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    });

    applyIncomingExecutionEvent(mockConversation.id, {
      event_id: "evt_1",
      execution_id: "exec_1",
      conversation_id: mockConversation.id,
      trace_id: "tr_exec_1",
      sequence: 1,
      queue_index: 0,
      type: "execution_started",
      timestamp: new Date().toISOString(),
      payload: {}
    });

    applyIncomingExecutionEvent(mockConversation.id, {
      event_id: "evt_2",
      execution_id: "exec_1",
      conversation_id: mockConversation.id,
      trace_id: "tr_exec_1",
      sequence: 2,
      queue_index: 0,
      type: "diff_generated",
      timestamp: new Date().toISOString(),
      payload: {
        diff: [
          {
            id: "diff_1",
            path: "src/main.ts",
            change_type: "modified",
            summary: "updated"
          }
        ]
      }
    });

    applyIncomingExecutionEvent(mockConversation.id, {
      event_id: "evt_3",
      execution_id: "exec_1",
      conversation_id: mockConversation.id,
      trace_id: "tr_exec_1",
      sequence: 3,
      queue_index: 0,
      type: "execution_done",
      timestamp: new Date().toISOString(),
      payload: {
        content: "done"
      }
    });

    expect(runtime.executions[0]?.state).toBe("completed");
    expect(runtime.diff.length).toBe(1);
    expect(runtime.messages[runtime.messages.length - 1]?.content).toContain("done");
  });

  it("stop calls backend stop endpoint", async () => {
    const runtime = ensureConversationRuntime(mockConversation, true);
    runtime.executions.push({
      id: "exec_running",
      workspace_id: "ws_local",
      conversation_id: mockConversation.id,
      message_id: "msg_running",
      state: "executing",
      mode: "agent",
      model_id: "gpt-5.3",
      mode_snapshot: "agent",
      model_snapshot: {
        model_id: "gpt-5.3"
      },
      project_revision_snapshot: 0,
      queue_index: 0,
      trace_id: "tr_running",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    });

    await stopConversationExecution(mockConversation);

    const stopCalls = fetchMock.mock.calls.filter(([url, init]) => {
      return String(url).endsWith(`/v1/conversations/${mockConversation.id}/stop`) && (init?.method ?? "GET") === "POST";
    });
    expect(stopCalls.length).toBe(1);
  });

  it("rollback restores execution states from snapshot point", async () => {
    ensureConversationRuntime(mockConversation, true);
    setConversationDraft(mockConversation.id, "first message");
    await submitConversationMessage(mockConversation, true);
    setConversationDraft(mockConversation.id, "second message");
    await submitConversationMessage(mockConversation, true);

    const runtime = ensureConversationRuntime(mockConversation, true);
    const secondUserMessage = [...runtime.messages].reverse().find((message) => message.role === "user");
    expect(secondUserMessage).toBeTruthy();

    const firstExecution = runtime.executions[0];
    expect(firstExecution).toBeTruthy();
    expect(firstExecution?.state).toBe("pending");

    if (firstExecution) {
      firstExecution.state = "completed";
    }
    runtime.executions.push({
      id: "exec_extra",
      workspace_id: "ws_local",
      conversation_id: mockConversation.id,
      message_id: "msg_extra",
      state: "queued",
      mode: "agent",
      model_id: "gpt-5.3",
      mode_snapshot: "agent",
      model_snapshot: {
        model_id: "gpt-5.3"
      },
      project_revision_snapshot: 0,
      queue_index: 9,
      trace_id: "tr_exec_extra",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    });

    await rollbackConversationToMessage(mockConversation.id, secondUserMessage!.id);

    expect(runtime.executions.length).toBe(1);
    expect(runtime.executions[0]?.id).toBe(firstExecution?.id);
    expect(runtime.executions[0]?.state).toBe("pending");
    expect(runtime.executions[0]?.queue_index).toBe(0);
  });

  it("caps runtime events to prevent unbounded growth", () => {
    ensureConversationRuntime(mockConversation, true);
    const runtime = ensureConversationRuntime(mockConversation, true);

    for (let index = 0; index < 1010; index += 1) {
      applyIncomingExecutionEvent(mockConversation.id, {
        event_id: `evt_cap_${index}`,
        execution_id: "exec_cap",
        conversation_id: mockConversation.id,
        trace_id: "tr_cap",
        sequence: index,
        queue_index: 0,
        type: "thinking_delta",
        timestamp: new Date(Date.now() + index).toISOString(),
        payload: { stage: "model_call", turn: index }
      });
    }

    expect(runtime.events.length).toBe(1000);
    expect(runtime.events[0]?.event_id).toBe("evt_cap_10");
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
