import { describe, expect, it } from "vitest";

import { localizeComposerSuggestionDetails } from "@/modules/conversation/views/composerSuggestionDetails";
import type { ComposerSuggestion } from "@/shared/types/api";

describe("composerSuggestionDetails", () => {
  it("localizes built-in slash command detail", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/help",
        detail: "Show slash command help.",
        insert_text: "/help",
        replace_start: 0,
        replace_end: 5
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => {
      if (key === "conversation.composer.suggestion.command.help") {
        return "显示斜杠命令帮助。";
      }
      return key;
    });

    expect(localized[0]?.detail).toBe("显示斜杠命令帮助。");
  });

  it("keeps backend detail for unknown command", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/custom",
        detail: "Custom command description",
        insert_text: "/custom",
        replace_start: 0,
        replace_end: 7
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => key);
    expect(localized[0]?.detail).toBe("Custom command description");
  });

  it("localizes skill command detail when mapping exists", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/algorithmic-art",
        detail: "Create generative art.",
        insert_text: "/algorithmic-art",
        replace_start: 0,
        replace_end: 16
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => {
      if (key === "conversation.composer.suggestion.command.algorithmicArt") {
        return "创建算法艺术（p5.js、可控随机与参数探索）。";
      }
      return key;
    });

    expect(localized[0]?.detail).toBe("创建算法艺术（p5.js、可控随机与参数探索）。");
  });

  it("does not render detail for file suggestions", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "resource",
        label: "@file:src/main.ts",
        detail: "will be removed",
        insert_text: "@file:src/main.ts",
        replace_start: 0,
        replace_end: 16
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => key);
    expect(localized[0]?.detail).toBe("");
  });

  it("fills localized detail for resource type suggestions", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "resource_type",
        label: "@rule:",
        insert_text: "@rule:",
        replace_start: 0,
        replace_end: 1
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => {
      if (key === "conversation.composer.suggestion.type.rule") {
        return "规则配置";
      }
      return key;
    });

    expect(localized[0]?.detail).toBe("规则配置");
  });
});
