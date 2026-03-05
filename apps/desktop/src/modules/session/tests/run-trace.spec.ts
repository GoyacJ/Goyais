import { describe, expect, it } from "vitest";

import { buildRunTraceViewModels } from "@/modules/session/views/processTrace";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

const baseExecution: Run = {
  id: "exec_trace_1",
  workspace_id: "ws_local",
  session_id: "conv_trace_1",
  message_id: "msg_trace_1",
  state: "executing",
  mode: "default",
  model_id: "gpt-5.3",
  mode_snapshot: "default",
  model_snapshot: {
    model_id: "gpt-5.3"
  },
  tokens_in: 20,
  tokens_out: 10,
  queue_index: 0,
  trace_id: "tr_trace_1",
  project_revision_snapshot: 0,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:20Z"
};

const baseEvent: RunLifecycleEvent = {
  event_id: "evt_trace_1",
  run_id: "exec_trace_1",
  session_id: "conv_trace_1",
  trace_id: "tr_trace_1",
  sequence: 1,
  queue_index: 0,
  type: "execution_started",
  timestamp: "2026-02-24T00:00:00Z",
  payload: {}
};

describe("run trace view model", () => {
  it("groups events by execution and builds readable summaries", () => {
    const events: RunLifecycleEvent[] = [
      baseEvent,
      {
        ...baseEvent,
        event_id: "evt_trace_2",
        sequence: 2,
        type: "thinking_delta",
        timestamp: "2026-02-24T00:00:03Z",
        payload: {
          stage: "model_call",
          delta: "model_call"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_3",
        sequence: 3,
        type: "thinking_delta",
        timestamp: "2026-02-24T00:00:05Z",
        payload: {
          stage: "assistant_output",
          delta: "<think>analyzing project structure and next command</think>"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_4",
        sequence: 4,
        type: "tool_call",
        timestamp: "2026-02-24T00:00:08Z",
        payload: {
          name: "run_command",
          risk_level: "low",
          input: { command: "ls -la" }
        }
      },
      {
        ...baseEvent,
        event_id: "evt_trace_5",
        sequence: 5,
        type: "tool_result",
        timestamp: "2026-02-24T00:00:10Z",
        payload: {
          name: "run_command",
          ok: true,
          output: "total 8\ndrwxr-xr-x"
        }
      }
    ];

    const traces = buildRunTraceViewModels(events, [baseExecution], "zh-CN", new Date("2026-02-24T00:00:15Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.executionId).toBe("exec_trace_1");
    expect(traces[0]?.summaryPrimary).toContain("已思考");
    expect(traces[0]?.summaryPrimary).toContain("调用 1 个工具");
    expect(traces[0]?.summarySecondary).toContain("Token in 20 / out 10 / total 30");
    expect(traces[0]?.summarySecondary).toContain("消息执行");
    expect(traces[0]?.summaryTone).toBe("primary");
    expect(traces[0]?.steps).toHaveLength(4);
    expect(traces[0]?.steps[1]?.title).toBe("思考");
    expect(traces[0]?.steps[1]?.summary).toContain("analyzing project structure");
    expect(traces[0]?.steps[2]?.summary).toContain("执行命令");
    expect(traces[0]?.steps[2]?.detail).toContain("工具");
  });

  it("renders basic detail level without raw payload", () => {
    const execution: Run = {
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
    const events: RunLifecycleEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_trace_basic_call",
        run_id: "exec_trace_basic",
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
        run_id: "exec_trace_basic",
        sequence: 2,
        type: "tool_result",
        payload: {
          name: "read_file",
          ok: false,
          error: "permission denied"
        }
      }
    ];

    const traces = buildRunTraceViewModels(events, [execution], "zh-CN", new Date("2026-02-24T00:01:02Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.state).toBe("completed");
    expect(traces[0]?.summaryTone).toBe("error");
    expect(traces[0]?.steps[0]?.rawPayload).toBe("");
    expect(traces[0]?.steps[0]?.summary).toContain("读取 README.md");
    expect(traces[0]?.steps[1]?.summary).toContain("失败");
  });

  it("handles failed execution summary with failed tool count", () => {
    const execution: Run = {
      ...baseExecution,
      id: "exec_trace_failed",
      message_id: "msg_trace_failed",
      state: "failed",
      updated_at: "2026-02-24T00:02:00Z"
    };
    const events: RunLifecycleEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_trace_failed_start",
        run_id: "exec_trace_failed",
        type: "execution_started",
        timestamp: "2026-02-24T00:01:40Z",
        payload: {}
      },
      {
        ...baseEvent,
        event_id: "evt_trace_failed_result",
        run_id: "exec_trace_failed",
        sequence: 2,
        type: "tool_result",
        timestamp: "2026-02-24T00:01:50Z",
        payload: {
          name: "run_command",
          ok: false
        }
      }
    ];

    const traces = buildRunTraceViewModels(events, [execution], "zh-CN", new Date("2026-02-24T00:02:05Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.summaryPrimary).toContain("执行失败");
    expect(traces[0]?.summaryPrimary).toContain("失败 1");
    expect(traces[0]?.summaryTone).toBe("error");
  });

  it("renders no-token secondary summary when usage is missing", () => {
    const execution: Run = {
      ...baseExecution,
      id: "exec_trace_no_usage",
      message_id: "msg_trace_no_usage",
      tokens_in: undefined,
      tokens_out: undefined
    };
    const traces = buildRunTraceViewModels([baseEvent], [execution], "zh-CN", new Date("2026-02-24T00:00:03Z"));
    expect(traces).toHaveLength(1);
    expect(traces[0]?.summarySecondary).toContain("消息执行");
    expect(traces[0]?.summarySecondary).not.toContain("Token");
    expect(traces[0]?.summaryTone).toBe("primary");
  });
});
