import { describe, expect, it } from "vitest";

import { normalizeExecutionEventsByExecution } from "@/modules/session/trace/normalize";
import type { RunLifecycleEvent } from "@/shared/types/api";

describe("run trace normalize", () => {
  it("sorts by sequence and timestamp and maps stage", () => {
    const events: RunLifecycleEvent[] = [
      {
        event_id: "evt_2",
        run_id: "exec_1",
        session_id: "conv_1",
        trace_id: "tr_1",
        sequence: 2,
        queue_index: 0,
        type: "thinking_delta",
        timestamp: "2026-02-24T00:00:02Z",
        payload: {
          stage: "assistant_output",
          delta: "Based on the project setup, I will check lint scripts first."
        }
      },
      {
        event_id: "evt_1",
        run_id: "exec_1",
        session_id: "conv_1",
        trace_id: "tr_1",
        sequence: 1,
        queue_index: 0,
        type: "tool_call",
        timestamp: "2026-02-24T00:00:01Z",
        payload: {
          name: "Bash",
          call_id: "call_1",
          input: {
            command: "echo hello"
          }
        }
      }
    ];

    const grouped = normalizeExecutionEventsByExecution(events);
    const trace = grouped.get("exec_1") ?? [];

    expect(trace).toHaveLength(2);
    expect(trace[0]?.id).toBe("evt_1");
    expect(trace[1]?.id).toBe("evt_2");
    expect(trace[1]?.stage).toBe("assistant_output");
    expect(trace[0]?.operationIntentKind).toBe("command");
    expect(trace[0]?.operationIntentValue).toBe("echo hello");
    expect(trace[1]?.reasoningSentence).toContain("I will check lint scripts first");
  });

  it("redacts sensitive payload keys", () => {
    const events: RunLifecycleEvent[] = [
      {
        event_id: "evt_sensitive",
        run_id: "exec_sensitive",
        session_id: "conv_1",
        trace_id: "tr_1",
        sequence: 1,
        queue_index: 0,
        type: "tool_call",
        timestamp: "2026-02-24T00:00:01Z",
        payload: {
          name: "Bash",
          input: {
            api_key: "secret-value",
            command: "ls"
          }
        }
      }
    ];

    const grouped = normalizeExecutionEventsByExecution(events);
    const trace = grouped.get("exec_sensitive") ?? [];

    expect(trace[0]?.payload.input).toEqual({ api_key: "***", command: "ls" });
    expect(trace[0]?.rawPayload).toContain("***");
    expect(trace[0]?.rawPayload).not.toContain("secret-value");
  });

  it("does not fallback reasoning sentence to stage placeholders", () => {
    const events: RunLifecycleEvent[] = [
      {
        event_id: "evt_placeholder",
        run_id: "exec_placeholder",
        session_id: "conv_1",
        trace_id: "tr_1",
        sequence: 1,
        queue_index: 0,
        type: "thinking_delta",
        timestamp: "2026-02-24T00:00:01Z",
        payload: {
          stage: "model_call",
          delta: "model_call"
        }
      }
    ];

    const grouped = normalizeExecutionEventsByExecution(events);
    const trace = grouped.get("exec_placeholder") ?? [];

    expect(trace[0]?.reasoningSentence).toBe("");
  });
});
