import { describe, expect, it } from "vitest";

import { toExecutionEventFromStreamPayload } from "@/modules/conversation/store/runEventAdapter";

describe("run event adapter", () => {
  it("maps run_started payload to execution_started event", () => {
    const result = toExecutionEventFromStreamPayload(
      {
        type: "run_started",
        session_id: "conv_adapter_1",
        run_id: "run_adapter_1",
        sequence: 9,
        timestamp: "2026-02-25T12:00:00Z",
        payload: {
          source: "worker"
        }
      },
      "conv_fallback"
    );

    expect(result).toBeTruthy();
    expect(result?.type).toBe("execution_started");
    expect(result?.conversation_id).toBe("conv_adapter_1");
    expect(result?.execution_id).toBe("run_adapter_1");
    expect(result?.sequence).toBe(9);
  });

  it("maps run_output_delta diff payload to diff_generated", () => {
    const result = toExecutionEventFromStreamPayload(
      {
        type: "run_output_delta",
        session_id: "conv_adapter_2",
        run_id: "run_adapter_2",
        sequence: 3,
        timestamp: "2026-02-25T12:01:00Z",
        payload: {
          diff: [
            { id: "diff_1", path: "README.md", change_type: "modified", summary: "updated" }
          ]
        }
      },
      "conv_fallback"
    );

    expect(result).toBeTruthy();
    expect(result?.type).toBe("diff_generated");
    expect(result?.conversation_id).toBe("conv_adapter_2");
    expect(result?.execution_id).toBe("run_adapter_2");
  });

  it("does not map run_output_delta without payload.diff to diff_generated", () => {
    const result = toExecutionEventFromStreamPayload(
      {
        type: "run_output_delta",
        session_id: "conv_adapter_3",
        run_id: "run_adapter_3",
        sequence: 4,
        timestamp: "2026-02-25T12:01:30Z",
        payload: {
          name: "Write",
          call_id: "call_1",
          output: "{\"ok\":true}"
        }
      },
      "conv_fallback"
    );

    expect(result).toBeTruthy();
    expect(result?.type).toBe("tool_result");
  });

  it("maps run_output_delta payload with explicit event_type to change_set event", () => {
    const result = toExecutionEventFromStreamPayload(
      {
        type: "run_output_delta",
        session_id: "conv_adapter_4",
        run_id: "run_adapter_4",
        sequence: 5,
        timestamp: "2026-02-25T12:01:40Z",
        payload: {
          event_type: "change_set_updated",
          change_set_id: "cs_123"
        }
      },
      "conv_fallback"
    );

    expect(result).toBeTruthy();
    expect(result?.type).toBe("change_set_updated");
    expect(result?.payload.change_set_id).toBe("cs_123");
  });

  it("returns null for legacy execution event payloads", () => {
    const result = toExecutionEventFromStreamPayload(
      {
        event_id: "evt_legacy_1",
        execution_id: "exec_legacy_1",
        conversation_id: "conv_legacy_1",
        trace_id: "tr_legacy_1",
        sequence: 1,
        queue_index: 0,
        type: "thinking_delta",
        timestamp: "2026-02-25T12:02:00Z",
        payload: {
          stage: "model_call"
        }
      },
      "conv_fallback"
    );

    expect(result).toBeNull();
  });
});
