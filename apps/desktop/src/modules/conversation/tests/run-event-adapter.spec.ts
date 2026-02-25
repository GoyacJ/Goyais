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
          files: 1,
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

  it("passes through legacy execution event payloads", () => {
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

    expect(result).toBeTruthy();
    expect(result?.event_id).toBe("evt_legacy_1");
    expect(result?.type).toBe("thinking_delta");
    expect(result?.execution_id).toBe("exec_legacy_1");
    expect(result?.conversation_id).toBe("conv_legacy_1");
  });
});
