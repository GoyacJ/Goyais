import { describe, expect, it } from "vitest";

import {
  buildRunTraceViewModelData,
  buildRunningActionBaseViewModelData,
  hydrateRunningActionElapsed
} from "@/modules/session/trace/present";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

const baseExecution: Run = {
  id: "exec_present_1",
  workspace_id: "ws_local",
  session_id: "conv_present_1",
  message_id: "msg_present_1",
  state: "executing",
  mode: "default",
  model_id: "gpt-5.3",
  mode_snapshot: "default",
  model_snapshot: {
    model_id: "gpt-5.3"
  },
  queue_index: 0,
  trace_id: "tr_present_1",
  project_revision_snapshot: 0,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:10Z"
};

const events: RunLifecycleEvent[] = [
  {
    event_id: "evt_present_model",
    run_id: "exec_present_1",
    session_id: "conv_present_1",
    trace_id: "tr_present_1",
    sequence: 1,
    queue_index: 0,
    type: "thinking_delta",
    timestamp: "2026-02-24T00:00:00Z",
    payload: {
      stage: "assistant_output",
      delta: "Planning lint configuration updates."
    }
  },
  {
    event_id: "evt_present_tool",
    run_id: "exec_present_1",
    session_id: "conv_present_1",
    trace_id: "tr_present_1",
    sequence: 2,
    queue_index: 0,
    type: "tool_call",
    timestamp: "2026-02-24T00:00:01Z",
    payload: {
      call_id: "call_1",
      name: "Bash",
      input: {
        command: "pnpm lint"
      },
      risk_level: "high"
    }
  }
];

describe("run trace present", () => {
  it("builds localized trace view models", () => {
    const zh = buildRunTraceViewModelData(events, [baseExecution], "zh-CN", new Date("2026-02-24T00:00:03Z"));
    const en = buildRunTraceViewModelData(events, [baseExecution], "en-US", new Date("2026-02-24T00:00:03Z"));

    expect(zh[0]?.summaryPrimary).toContain("调用 1 个工具");
    expect(en[0]?.summaryPrimary).toContain("tool calls");
    expect(zh[0]?.summaryTone).toBe("warning");
    expect(en[0]?.summaryTone).toBe("warning");
    expect(zh[0]?.steps[0]?.summary).toContain("Planning lint configuration updates.");
    expect(zh[0]?.steps[1]?.summary).toContain("执行命令 pnpm lint");
    expect(en[0]?.steps[1]?.summary).toContain("Run command pnpm lint");
  });

  it("does not expose raw payload for basic detail level", () => {
    const execution: Run = {
      ...baseExecution,
      id: "exec_present_basic",
      message_id: "msg_present_basic",
      agent_config_snapshot: {
        max_model_turns: 24,
        show_process_trace: true,
        trace_detail_level: "basic"
      }
    };
    const localizedEvents = events.map((event) => ({ ...event, run_id: "exec_present_basic" }));
    const traces = buildRunTraceViewModelData(localizedEvents, [execution], "zh-CN", new Date("2026-02-24T00:00:03Z"));

    expect(traces[0]?.steps[0]?.rawPayload).toBe("");
    expect(traces[0]?.steps[1]?.rawPayload).toBe("");
  });

  it("hydrates running action elapsed without rebuilding semantics", () => {
    const baseActions = buildRunningActionBaseViewModelData(events, [baseExecution], "zh-CN");
    const withElapsed = hydrateRunningActionElapsed(baseActions, "zh-CN", new Date("2026-02-24T00:00:05Z"));

    expect(withElapsed).toHaveLength(1);
    expect(withElapsed[0]?.primary).toBe("执行命令 pnpm lint");
    expect(withElapsed[0]?.secondary).toContain("操作");
    expect(withElapsed[0]?.elapsedLabel).toBe("4s");
  });

  it("hides non-meaningful model_call thinking steps", () => {
    const traces = buildRunTraceViewModelData(
      [
        {
          event_id: "evt_present_placeholder_model",
          run_id: "exec_present_1",
          session_id: "conv_present_1",
          trace_id: "tr_present_1",
          sequence: 0,
          queue_index: 0,
          type: "thinking_delta",
          timestamp: "2026-02-24T00:00:00Z",
          payload: {
            stage: "model_call",
            delta: "model_call"
          }
        },
        ...events
      ],
      [baseExecution],
      "zh-CN",
      new Date("2026-02-24T00:00:03Z")
    );

    expect(traces[0]?.steps.some((step) => step.summary.includes("model_call"))).toBe(false);
    expect(traces[0]?.steps).toHaveLength(2);
  });

  it("maps summary tone by execution state and trace signals", () => {
    const completedExecution: Run = {
      ...baseExecution,
      id: "exec_present_completed",
      message_id: "msg_present_completed",
      state: "completed",
      updated_at: "2026-02-24T00:00:12Z"
    };
    const completedEvents: RunLifecycleEvent[] = [
      {
        ...events[1]!,
        event_id: "evt_present_completed_call",
        run_id: "exec_present_completed",
        payload: {
          call_id: "call_completed_1",
          name: "Read",
          input: {
            path: "README.md"
          },
          risk_level: "low"
        }
      },
      {
        ...events[1]!,
        event_id: "evt_present_completed_result",
        run_id: "exec_present_completed",
        sequence: 3,
        type: "tool_result",
        payload: {
          call_id: "call_completed_1",
          name: "Read",
          ok: true
        }
      }
    ];

    const failedExecution: Run = {
      ...baseExecution,
      id: "exec_present_failed",
      message_id: "msg_present_failed",
      state: "failed",
      updated_at: "2026-02-24T00:00:12Z"
    };
    const failedEvents: RunLifecycleEvent[] = [
      {
        ...events[1]!,
        event_id: "evt_present_failed_result",
        run_id: "exec_present_failed",
        sequence: 1,
        type: "tool_result",
        payload: {
          name: "Bash",
          ok: false,
          error: "permission denied"
        }
      }
    ];

    const confirmingExecution: Run = {
      ...baseExecution,
      id: "exec_present_confirming",
      message_id: "msg_present_confirming",
      state: "confirming",
      updated_at: "2026-02-24T00:00:12Z"
    };
    const confirmingEvents: RunLifecycleEvent[] = [
      {
        ...events[0]!,
        event_id: "evt_present_approval",
        run_id: "exec_present_confirming",
        sequence: 1,
        payload: {
          stage: "run_approval_needed",
          call_id: "call_approval",
          name: "Bash",
          reason: "needs elevated privileges"
        }
      }
    ];

    const cancelledExecution: Run = {
      ...baseExecution,
      id: "exec_present_cancelled",
      message_id: "msg_present_cancelled",
      state: "cancelled",
      updated_at: "2026-02-24T00:00:12Z"
    };
    const cancelledEvents: RunLifecycleEvent[] = [
      {
        ...events[1]!,
        event_id: "evt_present_cancelled_call",
        run_id: "exec_present_cancelled",
        sequence: 1,
        payload: {
          call_id: "call_cancelled",
          name: "Read",
          input: {
            path: "README.md"
          },
          risk_level: "low"
        }
      }
    ];

    const traces = buildRunTraceViewModelData(
      [...completedEvents, ...failedEvents, ...confirmingEvents, ...cancelledEvents],
      [completedExecution, failedExecution, confirmingExecution, cancelledExecution],
      "en-US",
      new Date("2026-02-24T00:00:13Z")
    );

    expect(traces.find((trace) => trace.executionId === "exec_present_completed")?.summaryTone).toBe("success");
    expect(traces.find((trace) => trace.executionId === "exec_present_failed")?.summaryTone).toBe("error");
    expect(traces.find((trace) => trace.executionId === "exec_present_confirming")?.summaryTone).toBe("warning");
    expect(traces.find((trace) => trace.executionId === "exec_present_cancelled")?.summaryTone).toBe("neutral");
  });
});
