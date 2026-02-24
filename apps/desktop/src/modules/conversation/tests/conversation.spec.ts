import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import {
  applyIncomingExecutionEvent,
  ensureConversationRuntime,
  rollbackConversationToMessage,
  resetConversationStore,
  setConversationDraft,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import MainConversationPanel from "@/modules/conversation/components/MainConversationPanel.vue";
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
    runtime.messages.push({
      id: "msg_1",
      conversation_id: mockConversation.id,
      role: "user",
      content: "执行任务",
      queue_index: 0,
      created_at: new Date().toISOString()
    });
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

  it("does not append duplicate terminal message for replayed execution_done", () => {
    const runtime = ensureConversationRuntime(mockConversation, true);
    runtime.messages.push({
      id: "msg_user_0",
      conversation_id: mockConversation.id,
      role: "user",
      content: "查看当前项目",
      queue_index: 0,
      created_at: new Date().toISOString()
    });
    runtime.executions.push({
      id: "exec_done_once",
      workspace_id: "ws_local",
      conversation_id: mockConversation.id,
      message_id: "msg_user_0",
      state: "executing",
      mode: "agent",
      model_id: "gpt-5.3",
      mode_snapshot: "agent",
      model_snapshot: {
        model_id: "gpt-5.3"
      },
      project_revision_snapshot: 0,
      queue_index: 0,
      trace_id: "tr_done_once",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    });

    const doneEvent = {
      event_id: "evt_done_once",
      execution_id: "exec_done_once",
      conversation_id: mockConversation.id,
      trace_id: "tr_done_once",
      sequence: 9,
      queue_index: 0,
      type: "execution_done" as const,
      timestamp: new Date().toISOString(),
      payload: { content: "项目读取完成" }
    };

    applyIncomingExecutionEvent(mockConversation.id, doneEvent);
    applyIncomingExecutionEvent(mockConversation.id, doneEvent);

    const assistantMessages = runtime.messages.filter(
      (message) => message.role === "assistant" && message.content.includes("项目读取完成")
    );
    expect(assistantMessages).toHaveLength(1);
  });

  it("inserts terminal message by queue_index to keep message order stable", () => {
    const runtime = ensureConversationRuntime(mockConversation, true);
    runtime.messages.push(
      {
        id: "msg_user_0",
        conversation_id: mockConversation.id,
        role: "user",
        content: "第一条",
        queue_index: 0,
        created_at: "2026-02-24T00:00:00Z"
      },
      {
        id: "msg_user_1",
        conversation_id: mockConversation.id,
        role: "user",
        content: "第二条",
        queue_index: 1,
        created_at: "2026-02-24T00:00:01Z"
      }
    );
    runtime.executions.push({
      id: "exec_order_0",
      workspace_id: "ws_local",
      conversation_id: mockConversation.id,
      message_id: "msg_user_0",
      state: "executing",
      mode: "agent",
      model_id: "gpt-5.3",
      mode_snapshot: "agent",
      model_snapshot: {
        model_id: "gpt-5.3"
      },
      project_revision_snapshot: 0,
      queue_index: 0,
      trace_id: "tr_order_0",
      created_at: "2026-02-24T00:00:00Z",
      updated_at: "2026-02-24T00:00:00Z"
    });

    applyIncomingExecutionEvent(mockConversation.id, {
      event_id: "evt_order_done",
      execution_id: "exec_order_0",
      conversation_id: mockConversation.id,
      trace_id: "tr_order_0",
      sequence: 3,
      queue_index: 0,
      type: "execution_done",
      timestamp: "2026-02-24T00:00:02Z",
      payload: {
        content: "第一条已完成"
      }
    });

    const contents = runtime.messages.map((message) => message.content);
    expect(contents).toEqual(["第一条", "第一条已完成", "第二条"]);
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

  it("hides thinking hint when active trace summary is rendered", () => {
    const wrapper = mount(MainConversationPanel, {
      props: {
        messages: [
          {
            id: "msg_user_trace",
            conversation_id: mockConversation.id,
            role: "user",
            content: "查看当前项目",
            created_at: "2026-02-24T00:00:00Z",
            queue_index: 0
          }
        ],
        queuedCount: 0,
        pendingCount: 1,
        executingCount: 0,
        hasActiveExecution: true,
        activeTraceCount: 1,
        executionTraces: [
          {
            executionId: "exec_trace_1",
            messageId: "msg_user_trace",
            queueIndex: 0,
            state: "executing",
            isRunning: true,
            summary: "已思考 12s，已调用 2 个工具",
            isExpanded: true,
            steps: [
              {
                id: "trace_step_1",
                title: "思考",
                summary: "model_call",
                details: ""
              }
            ]
          }
        ],
        runningActions: [
          {
            actionId: "tool:exec_trace_1:call_1",
            executionId: "exec_trace_1",
            queueIndex: 0,
            type: "tool",
            label: "工具 read_file",
            startedAt: "2026-02-24T00:00:01Z",
            elapsedMs: 3000,
            elapsedLabel: "3s"
          }
        ],
        draft: "",
        mode: "agent",
        modelId: "gpt-5.3",
        placeholder: "输入消息",
        modelOptions: [{ value: "gpt-5.3", label: "GPT-5.3" }]
      }
    });

    expect(wrapper.find(".execution-hint").exists()).toBe(false);
    expect(wrapper.find(".trace-summary-inline").exists()).toBe(true);
    expect(wrapper.find(".trace-running-line").exists()).toBe(true);
  });

  it("renders trace disclosure without legacy trace card class", () => {
    const wrapper = mount(MainConversationPanel, {
      props: {
        messages: [
          {
            id: "msg_user_trace_2",
            conversation_id: mockConversation.id,
            role: "user",
            content: "执行任务",
            created_at: "2026-02-24T00:00:00Z",
            queue_index: 1
          }
        ],
        queuedCount: 0,
        pendingCount: 0,
        executingCount: 0,
        hasActiveExecution: false,
        activeTraceCount: 0,
        executionTraces: [
          {
            executionId: "exec_trace_2",
            messageId: "msg_user_trace_2",
            queueIndex: 1,
            state: "completed",
            isRunning: false,
            summary: "已思考 8s，调用 1 个工具",
            isExpanded: false,
            steps: [
              {
                id: "trace_step_2",
                title: "工具调用",
                summary: "read_file (low)",
                details: ""
              }
            ]
          }
        ],
        runningActions: [],
        draft: "",
        mode: "agent",
        modelId: "gpt-5.3",
        placeholder: "输入消息",
        modelOptions: [{ value: "gpt-5.3", label: "GPT-5.3" }]
      }
    });

    expect(wrapper.find("details.trace-disclosure").exists()).toBe(true);
    expect(wrapper.find(".trace-item").exists()).toBe(false);
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
