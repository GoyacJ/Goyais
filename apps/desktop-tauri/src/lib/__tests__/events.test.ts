import { describe, expect, it } from "vitest";

import { normalizeEventEnvelope } from "@/lib/events";
import type { EventEnvelope } from "@/types/generated";

describe("normalizeEventEnvelope", () => {
  it("marks tool_call requiring confirmation as waiting_confirmation", () => {
    const event: EventEnvelope = {
      protocol_version: "2.0.0",
      trace_id: "trace-1",
      event_id: "e-1",
      run_id: "r-1",
      seq: 2,
      ts: "2026-02-20T10:00:00Z",
      type: "tool_call",
      payload: {
        trace_id: "trace-1",
        call_id: "c-1",
        tool_name: "run_command",
        args: { cmd: "pnpm test" },
        requires_confirmation: true
      }
    };

    const result = normalizeEventEnvelope(event);
    expect(result.streamState).toBe("waiting_confirmation");
    expect(result.summary).toContain("run_command");
  });

  it("marks done event as completed", () => {
    const event: EventEnvelope = {
      protocol_version: "2.0.0",
      trace_id: "trace-1",
      event_id: "e-2",
      run_id: "r-1",
      seq: 3,
      ts: "2026-02-20T10:00:01Z",
      type: "done",
      payload: { trace_id: "trace-1", status: "completed", message: "ok" }
    };

    const result = normalizeEventEnvelope(event);
    expect(result.streamState).toBe("completed");
  });
});
