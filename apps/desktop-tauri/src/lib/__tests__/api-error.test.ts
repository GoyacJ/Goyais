import { describe, expect, it } from "vitest";

import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";

describe("api-error", () => {
  it("parses goyais error shape", async () => {
    const response = new Response(
      JSON.stringify({
        error: {
          code: "E_INTERNAL",
          message: "server exploded",
          trace_id: "trace-1",
          retryable: true
        }
      }),
      {
      status: 503,
      headers: { "content-type": "application/json" }
      }
    );

    const error = await normalizeHttpError(response);

    expect(error.code).toBe("E_INTERNAL");
    expect(error.retryable).toBe(true);
    expect(error.message).toContain("server exploded");
    expect(error.traceId).toBe("trace-1");
  });

  it("normalizes unknown runtime error", () => {
    const error = normalizeUnknownError(new Error("network timeout"));
    expect(error.code).toBe("NETWORK_OR_RUNTIME_ERROR");
    expect(error.retryable).toBe(true);
  });
});
