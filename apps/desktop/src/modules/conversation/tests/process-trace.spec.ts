import { describe, expect, it } from "vitest";

import { buildProcessTraceItems } from "@/modules/conversation/views/processTrace";
import type { ExecutionEvent } from "@/shared/types/api";

const baseEvent: ExecutionEvent = {
  event_id: "evt_1",
  execution_id: "exec_trace_1",
  conversation_id: "conv_trace_1",
  trace_id: "tr_trace_1",
  sequence: 1,
  queue_index: 0,
  type: "execution_started",
  timestamp: "2026-02-24T00:00:00Z",
  payload: {}
};

describe("process trace rendering", () => {
  it("filters by execution_id and includes verbose details", () => {
    const events: ExecutionEvent[] = [
      baseEvent,
      {
        ...baseEvent,
        event_id: "evt_2",
        sequence: 2,
        type: "thinking_delta",
        payload: {
          stage: "model_call",
          delta: "analyzing current project"
        }
      },
      {
        ...baseEvent,
        event_id: "evt_other",
        execution_id: "exec_trace_other",
        sequence: 3,
        type: "tool_call",
        payload: {
          name: "read_file"
        }
      }
    ];

    const items = buildProcessTraceItems(events, "exec_trace_1", "verbose");
    expect(items.length).toBe(2);
    expect(items[0]?.title).toBe("Execution Started");
    expect(items[1]?.summary).toContain("model_call");
    expect(items[1]?.details).toContain("analyzing current project");
  });

  it("uses compact summary when detail level is basic", () => {
    const events: ExecutionEvent[] = [
      {
        ...baseEvent,
        event_id: "evt_tool_call",
        sequence: 10,
        type: "tool_call",
        payload: {
          name: "run_command",
          risk_level: "high",
          input: {
            command: "npm install"
          }
        }
      },
      {
        ...baseEvent,
        event_id: "evt_tool_result",
        sequence: 11,
        type: "tool_result",
        payload: {
          name: "run_command",
          ok: true,
          output: {
            exit_code: 0
          }
        }
      }
    ];

    const items = buildProcessTraceItems(events, "exec_trace_1", "basic");
    expect(items.length).toBe(2);
    expect(items[0]?.summary).toContain("run_command");
    expect(items[0]?.details).toBe("");
    expect(items[1]?.summary).toContain("done");
  });
});
