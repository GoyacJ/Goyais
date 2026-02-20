import { describe, expect, it } from "vitest";
import { getValidationErrors, validateEventEnvelope } from "../src/validators";

describe("event envelope schema", () => {
  it("accepts valid tool_call event", () => {
    const validEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_1",
      run_id: "run_1",
      seq: 1,
      ts: "2026-02-20T00:00:00Z",
      type: "tool_call",
      payload: {
        trace_id: "trace_1",
        call_id: "call_1",
        tool_name: "read_file",
        args: { path: "README.md" },
        requires_confirmation: false
      }
    };

    expect(validateEventEnvelope(validEvent)).toBe(true);
    expect(getValidationErrors()).toEqual([]);
  });

  it("rejects invalid tool_call event when call_id is missing", () => {
    const invalidEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_2",
      run_id: "run_1",
      seq: 2,
      ts: "2026-02-20T00:00:01Z",
      type: "tool_call",
      payload: {
        trace_id: "trace_1",
        tool_name: "read_file",
        args: { path: "README.md" },
        requires_confirmation: false
      }
    };

    expect(validateEventEnvelope(invalidEvent)).toBe(false);
    expect(getValidationErrors().join(" ")).toContain("must have required property 'call_id'");
  });

  it("rejects plan event without summary", () => {
    const invalidEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_3",
      run_id: "run_1",
      seq: 3,
      ts: "2026-02-20T00:00:02Z",
      type: "plan",
      payload: {
        trace_id: "trace_1",
        steps: ["read", "patch"]
      }
    };

    expect(validateEventEnvelope(invalidEvent)).toBe(false);
    expect(getValidationErrors().join(" ")).toContain("must have required property 'summary'");
  });

  it("rejects error event without goyais error object", () => {
    const invalidEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_4",
      run_id: "run_1",
      seq: 4,
      ts: "2026-02-20T00:00:03Z",
      type: "error",
      payload: {
        trace_id: "trace_1",
        message: "legacy payload"
      }
    };

    expect(validateEventEnvelope(invalidEvent)).toBe(false);
    expect(getValidationErrors().join(" ")).toContain("must have required property 'error'");
  });

  it("rejects done event with invalid status", () => {
    const invalidEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_5",
      run_id: "run_1",
      seq: 5,
      ts: "2026-02-20T00:00:04Z",
      type: "done",
      payload: {
        trace_id: "trace_1",
        status: "ok"
      }
    };

    expect(validateEventEnvelope(invalidEvent)).toBe(false);
    expect(getValidationErrors().join(" ")).toContain("must be equal to one of the allowed values");
  });

  it("rejects failed tool_result without error", () => {
    const invalidEvent = {
      protocol_version: "2.0.0",
      trace_id: "trace_1",
      event_id: "evt_6",
      run_id: "run_1",
      seq: 6,
      ts: "2026-02-20T00:00:05Z",
      type: "tool_result",
      payload: {
        trace_id: "trace_1",
        call_id: "call_1",
        ok: false
      }
    };

    expect(validateEventEnvelope(invalidEvent)).toBe(false);
    expect(getValidationErrors().join(" ")).toContain("must have required property 'error'");
  });
});
