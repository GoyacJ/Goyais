import { describe, expect, it } from "vitest";

import {
  dedupeComposerSuggestions,
  resolveLocalSuggestionLimit,
  resolveSuggestionContext,
  shouldRequestRemoteSuggestions
} from "@/modules/conversation/views/composerSuggestionPolicy";
import type { ComposerCatalog, ComposerSuggestion } from "@/shared/types/api";

describe("composerSuggestionPolicy", () => {
  const baseCatalog: ComposerCatalog = {
    revision: "rev",
    commands: [
      { name: "help", description: "Show help", kind: "control" },
      { name: "clear", description: "Clear state", kind: "control" },
      { name: "cost", description: "Show cost", kind: "control" }
    ],
    resources: []
  };

  it("treats slash token as slash context and returns full command limit", () => {
    const context = resolveSuggestionContext("/", 1);
    expect(context.kind).toBe("slash");
    expect(context.query).toBe("");
    expect(context.tokenRange).toEqual({ start: 0, end: 1 });
    expect(resolveLocalSuggestionLimit(context, baseCatalog, 12)).toBe(baseCatalog.commands.length);
  });

  it("does not request remote suggestions for slash context", () => {
    const context = resolveSuggestionContext("/co", 3);
    expect(context.kind).toBe("slash");
    expect(shouldRequestRemoteSuggestions(context, 3)).toBe(false);
    expect(shouldRequestRemoteSuggestions(context, 0)).toBe(false);
  });

  it("forces remote suggestions for @file token", () => {
    const context = resolveSuggestionContext("@file:src", 9);
    expect(context.kind).toBe("resource");
    expect(context.isFileToken).toBe(true);
    expect(shouldRequestRemoteSuggestions(context, 0)).toBe(true);
  });

  it("keeps existing local+remote policy for non-file resources", () => {
    const context = resolveSuggestionContext("@rule:abc", 9);
    expect(context.kind).toBe("resource");
    expect(context.isFileToken).toBe(false);
    expect(resolveLocalSuggestionLimit(context, baseCatalog, 12)).toBe(12);
    expect(shouldRequestRemoteSuggestions(context, 0)).toBe(false);
    expect(shouldRequestRemoteSuggestions(context, 1)).toBe(true);
  });

  it("deduplicates suggestions by kind and insert_text", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/help",
        insert_text: "/help",
        replace_start: 0,
        replace_end: 1
      },
      {
        kind: "command",
        label: "/help",
        insert_text: "/help",
        replace_start: 0,
        replace_end: 1
      },
      {
        kind: "command",
        label: "/clear",
        insert_text: "/clear",
        replace_start: 0,
        replace_end: 1
      }
    ];

    const deduped = dedupeComposerSuggestions(suggestions);
    expect(deduped).toHaveLength(2);
    expect(deduped.map((item) => item.insert_text)).toEqual(["/help", "/clear"]);
  });
});
