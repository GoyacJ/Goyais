import { describe, expect, it } from "vitest";

import { buildRunningActionViewModels } from "@/modules/conversation/views/runningActions";
import type { Execution, ExecutionEvent } from "@/shared/types/api";

const baseExecution: Execution = {
  id: "exec_running_1",
  workspace_id: "ws_local",
  conversation_id: "conv_running_1",
  message_id: "msg_running_1",
  state: "executing",
  mode: "agent",
  model_id: "gpt-5.3",
  mode_snapshot: "agent",
  model_snapshot: {
    model_id: "gpt-5.3"
  },
  queue_index: 0,
  trace_id: "tr_running_1",
  project_revision_snapshot: 0,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:00Z"
};

const baseEvent: ExecutionEvent = {
  event_id: "evt_running_1",
  execution_id: "exec_running_1",
  conversation_id: "conv_running_1",
  trace_id: "tr_running_1",
  sequence: 1,
  queue_index: 0,
  type: "execution_started",
  timestamp: "2026-02-24T00:00:00Z",
  payload: {}
};

describe("running actions view", () => {
  it("shows concurrent running actions with elapsed time", () => {
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_model_call",
        type: "thinking_delta",
        sequence: 1,
        timestamp: "2026-02-24T00:00:00Z",
        payload: {
          stage: "model_call"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_call_a",
        type: "tool_call",
        sequence: 2,
        timestamp: "2026-02-24T00:00:01Z",
        payload: {
          call_id: "call_a",
          name: "run_command"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_call_b",
        type: "tool_call",
        sequence: 3,
        timestamp: "2026-02-24T00:00:02Z",
        payload: {
          call_id: "call_b",
          name: "read_file",
          input: {
            path: "README.md"
          }
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_result_a",
        type: "tool_result",
        sequence: 4,
        timestamp: "2026-02-24T00:00:03Z",
        payload: {
          call_id: "call_a",
          name: "run_command",
          ok: true
        }
      }
    ];

    const actions = buildRunningActionViewModels(events, [baseExecution], "zh-CN", new Date("2026-02-24T00:00:05Z"));
    expect(actions).toHaveLength(2);
    expect(actions.map((item) => item.primary)).toContain("模型推理");
    expect(actions.map((item) => item.primary)).toContain("工具 read_file");
    expect(actions.find((item) => item.primary === "工具 read_file")?.elapsedLabel).toBe("3s");
    expect(actions.find((item) => item.primary === "工具 read_file")?.secondary).toContain("path");
  });

  it("falls back to name+sequence matching when call_id is missing", () => {
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_tool_call_old",
        type: "tool_call",
        sequence: 1,
        timestamp: "2026-02-24T00:01:00Z",
        payload: {
          name: "read_file"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_result_old",
        type: "tool_result",
        sequence: 2,
        timestamp: "2026-02-24T00:01:01Z",
        payload: {
          name: "read_file",
          ok: true
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_call_new",
        type: "tool_call",
        sequence: 3,
        timestamp: "2026-02-24T00:01:02Z",
        payload: {
          name: "read_file"
        }
      }
    ];

    const actions = buildRunningActionViewModels(events, [baseExecution], "zh-CN", new Date("2026-02-24T00:01:05Z"));
    expect(actions).toHaveLength(1);
    expect(actions[0]?.primary).toBe("工具 read_file");
    expect(actions[0]?.elapsedLabel).toBe("3s");
  });

  it("shows approval action while execution is confirming", () => {
    const confirmingExecution: Execution = {
      ...baseExecution,
      id: "exec_confirming_1",
      state: "confirming"
    };
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        execution_id: "exec_confirming_1",
        event_id: "evt_approval_needed",
        type: "thinking_delta",
        sequence: 1,
        timestamp: "2026-02-24T00:02:00Z",
        payload: {
          stage: "run_approval_needed",
          call_id: "call_approval_1",
          name: "Bash",
          reason: "requires high-risk permission"
        }
      }
    ];

    const actions = buildRunningActionViewModels(events, [confirmingExecution], "zh-CN", new Date("2026-02-24T00:02:04Z"));
    expect(actions).toHaveLength(1);
    expect(actions[0]?.type).toBe("approval");
    expect(actions[0]?.primary).toBe("等待授权 Bash");
    expect(actions[0]?.elapsedLabel).toBe("4s");
  });

  it("localizes primary labels in english locale", () => {
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_tool_call_en",
        type: "tool_call",
        sequence: 1,
        timestamp: "2026-02-24T00:03:00Z",
        payload: {
          call_id: "call_en",
          name: "read_file"
        }
      }
    ];

    const actions = buildRunningActionViewModels(events, [baseExecution], "en-US", new Date("2026-02-24T00:03:03Z"));
    expect(actions).toHaveLength(1);
    expect(actions[0]?.primary).toBe("Tool read_file");
    expect(actions[0]?.elapsedLabel).toBe("3s");
  });
});
