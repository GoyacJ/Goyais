import { describe, expect, it } from "vitest";

import {
  extractOperationIntent,
  extractOperationSummary,
  extractReasoningSentence,
  extractResultSummary,
  redactSensitivePayload
} from "@/modules/session/trace/summarize";

describe("run trace summarize", () => {
  it("extracts one sentence from reasoning delta and strips think tags", () => {
    const sentence = extractReasoningSentence("<think>First sentence. Second sentence.</think>");

    expect(sentence).toBe("First sentence.");
  });

  it("returns empty reasoning sentence for placeholder-only content", () => {
    expect(extractReasoningSentence("model_call")).toBe("");
    expect(extractReasoningSentence("assistant_output")).toBe("");
    expect(extractReasoningSentence("thinking")).toBe("");
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

  it("extracts operation intent by key priority", () => {
    const intent = extractOperationIntent({
      input: {
        query: "foo",
        path: "README.md",
        command: "pnpm lint"
      }
    });

    expect(intent.kind).toBe("command");
    expect(intent.value).toBe("pnpm lint");
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
