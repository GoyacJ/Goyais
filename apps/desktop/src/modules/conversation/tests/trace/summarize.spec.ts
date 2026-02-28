import { describe, expect, it } from "vitest";

import {
  extractOperationSummary,
  extractReasoningSentence,
  extractResultSummary,
  redactSensitivePayload
} from "@/modules/conversation/trace/summarize";

describe("trace summarize", () => {
  it("extracts one sentence from reasoning delta and strips think tags", () => {
    const sentence = extractReasoningSentence(
      "<think>First sentence. Second sentence.</think>",
      "fallback"
    );

    expect(sentence).toBe("First sentence.");
  });

  it("extracts operation summary by key priority", () => {
    const summary = extractOperationSummary({
      input: {
        query: "foo",
        path: "README.md",
        command: "pnpm lint"
      }
    });

    expect(summary).toContain("command");
    expect(summary).toContain("pnpm lint");
  });

  it("extracts result summary from error when failed", () => {
    const summary = extractResultSummary(
      {
        ok: false,
        error: "permission denied. not allowed"
      },
      false
    );

    expect(summary).toContain("permission denied");
  });

  it("redacts nested sensitive keys", () => {
    const payload = redactSensitivePayload({
      token: "a",
      nested: {
        authorization: "b",
        value: "ok"
      }
    });

    expect(payload).toEqual({
      token: "***",
      nested: {
        authorization: "***",
        value: "ok"
      }
    });
  });
});
