import { describe, expect, it } from "vitest";

import {
  buildExecutionTraceViewModelData,
  buildRunningActionBaseViewModelData,
  hydrateRunningActionElapsed
} from "@/modules/conversation/trace/present";
import type { Execution, ExecutionEvent } from "@/shared/types/api";

const baseExecution: Execution = {
  id: "exec_present_1",
  workspace_id: "ws_local",
  conversation_id: "conv_present_1",
  message_id: "msg_present_1",
  state: "executing",
  mode: "agent",
  model_id: "gpt-5.3",
  mode_snapshot: "agent",
  model_snapshot: {
    model_id: "gpt-5.3"
  },
  queue_index: 0,
  trace_id: "tr_present_1",
  project_revision_snapshot: 0,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:10Z"
};

const events: ExecutionEvent[] = [
  {
    event_id: "evt_present_model",
    execution_id: "exec_present_1",
    conversation_id: "conv_present_1",
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
    execution_id: "exec_present_1",
    conversation_id: "conv_present_1",
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

describe("trace present", () => {
  it("builds localized trace view models", () => {
    const zh = buildExecutionTraceViewModelData(events, [baseExecution], "zh-CN", new Date("2026-02-24T00:00:03Z"));
    const en = buildExecutionTraceViewModelData(events, [baseExecution], "en-US", new Date("2026-02-24T00:00:03Z"));

    expect(zh[0]?.summaryPrimary).toContain("调用 1 个工具");
    expect(en[0]?.summaryPrimary).toContain("tool calls");
    expect(zh[0]?.steps[1]?.summary).toContain("高风险");
    expect(en[0]?.steps[1]?.summary).toContain("high risk");
  });

  it("does not expose raw payload for basic detail level", () => {
    const execution: Execution = {
      ...baseExecution,
      id: "exec_present_basic",
      message_id: "msg_present_basic",
      agent_config_snapshot: {
        max_model_turns: 24,
        show_process_trace: true,
        trace_detail_level: "basic"
      }
    };
    const localizedEvents = events.map((event) => ({ ...event, execution_id: "exec_present_basic" }));
    const traces = buildExecutionTraceViewModelData(localizedEvents, [execution], "zh-CN", new Date("2026-02-24T00:00:03Z"));

    expect(traces[0]?.steps[0]?.rawPayload).toBe("");
    expect(traces[0]?.steps[1]?.rawPayload).toBe("");
  });

  it("hydrates running action elapsed without rebuilding semantics", () => {
    const baseActions = buildRunningActionBaseViewModelData(events, [baseExecution], "zh-CN");
    const withElapsed = hydrateRunningActionElapsed(baseActions, "zh-CN", new Date("2026-02-24T00:00:05Z"));

    expect(withElapsed).toHaveLength(1);
    expect(withElapsed[0]?.primary).toBe("工具 Bash");
    expect(withElapsed[0]?.secondary).toContain("command");
    expect(withElapsed[0]?.elapsedLabel).toBe("4s");
  });
});
