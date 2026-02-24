import { describe, expect, it } from "vitest";

import { buildExecutionTraceViewModels } from "@/modules/conversation/views/processTrace";
import type { Execution, ExecutionEvent } from "@/shared/types/api";

const baseExecution: Execution = {
  id: "exec_trace_1",
  workspace_id: "ws_local",
  conversation_id: "conv_trace_1",
  message_id: "msg_trace_1",
  state: "executing",
  mode: "agent",
  model_id: "gpt-5.3",
  mode_snapshot: "agent",
  model_snapshot: {
    model_id: "gpt-5.3"
  },
  queue_index: 0,
  trace_id: "tr_trace_1",
  project_revision_snapshot: 0,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:20Z"
};

const baseEvent: ExecutionEvent = {
  event_id: "evt_trace_1",
  execution_id: "exec_trace_1",
  conversation_id: "conv_trace_1",
  trace_id: "tr_trace_1",
  sequence: 1,
  queue_index: 0,
  type: "execution_started",
  timestamp: "2026-02-24T00:00:00Z",
  payload: {}
};

describe("execution trace view model", () => {
  it("groups events by execution and builds natural summary", () => {
    const events: ExecutionEvent[] = [
      baseEvent,
      {
        ...baseEvent,
        event_id: "evt_trace_2",
        sequence: 2,
        type: "thinking_delta",
        timestamp: "2026-02-24T00:00:05Z",
        payload: {
          stage: "model_call",
          delta: "analyzing project structure"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_3",
        sequence: 3,
        type: "tool_call",
        timestamp: "2026-02-24T00:00:08Z",
        payload: {
          name: "run_command",
          risk_level: "low"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_4",
        sequence: 4,
        type: "tool_result",
        timestamp: "2026-02-24T00:00:10Z",
        payload: {
          name: "run_command",
          ok: true
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_other",
        execution_id: "exec_other",
        sequence: 1,
        type: "thinking_delta",
        payload: {
          stage: "model_call"
        }
      }
    ];

    const traces = buildExecutionTraceViewModels(events, [baseExecution], new Date("2026-02-24T00:00:15Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.executionId).toBe("exec_trace_1");
    expect(traces[0]?.summary).toContain("已思考");
    expect(traces[0]?.summary).toContain("调用 1 个工具");
    expect(traces[0]?.steps).toHaveLength(4);
    expect(traces[0]?.steps[1]?.title).toBe("思考");
  });

  it("renders basic detail level without payload details", () => {
    const execution: Execution = {
      ...baseExecution,
      id: "exec_trace_basic",
      message_id: "msg_trace_basic",
      state: "completed",
      updated_at: "2026-02-24T00:01:00Z",
      agent_config_snapshot: {
        max_model_turns: 24,
        show_process_trace: true,
        trace_detail_level: "basic"
      }
    };
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_trace_basic_call",
        execution_id: "exec_trace_basic",
        type: "tool_call",
        payload: {
          name: "read_file",
          risk_level: "low",
          input: { path: "README.md" }
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_basic_result",
        execution_id: "exec_trace_basic",
        sequence: 2,
        type: "tool_result",
        payload: {
          name: "read_file",
          ok: false
        }
      }
    ];

    const traces = buildExecutionTraceViewModels(events, [execution], new Date("2026-02-24T00:01:02Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.state).toBe("completed");
    expect(traces[0]?.steps[0]?.details).toBe("");
    expect(traces[0]?.steps[1]?.summary).toContain("failed");
  });

  it("handles failed execution summary with failed tool count", () => {
    const execution: Execution = {
      ...baseExecution,
      id: "exec_trace_failed",
      message_id: "msg_trace_failed",
      state: "failed",
      updated_at: "2026-02-24T00:02:00Z"
    };
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_trace_failed_start",
        execution_id: "exec_trace_failed",
        type: "execution_started",
        timestamp: "2026-02-24T00:01:40Z",
        payload: {}
      },
      {
        ...baseEvent,
        event_id: "evt_trace_failed_result",
        execution_id: "exec_trace_failed",
        sequence: 2,
        type: "tool_result",
        timestamp: "2026-02-24T00:01:50Z",
        payload: {
          name: "run_command",
          ok: false
        }
      }
    ];

    const traces = buildExecutionTraceViewModels(events, [execution], new Date("2026-02-24T00:02:05Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.summary).toContain("执行失败");
    expect(traces[0]?.summary).toContain("1 个失败");
  });
});
