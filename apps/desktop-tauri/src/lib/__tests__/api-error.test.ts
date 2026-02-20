import { describe, expect, it } from "vitest";

import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";

describe("api-error", () => {
  it("marks 5xx as retryable", async () => {
    const response = new Response(JSON.stringify({ detail: "server exploded" }), {
      status: 503,
      headers: { "content-type": "application/json" }
    });

    const error = await normalizeHttpError(response);

    expect(error.code).toBe("HTTP_503");
    expect(error.retryable).toBe(true);
    expect(error.message).toContain("server exploded");
  });

  it("normalizes unknown runtime error", () => {
    const error = normalizeUnknownError(new Error("network timeout"));
    expect(error.code).toBe("NETWORK_OR_RUNTIME_ERROR");
    expect(error.retryable).toBe(true);
  });
});
