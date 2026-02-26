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

  it("localizes requested skill commands in zh locale", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/antfu",
        detail: "English detail",
        insert_text: "/antfu",
        replace_start: 0,
        replace_end: 6
      },
      {
        kind: "command",
        label: "/brand-guidelines",
        detail: "English detail",
        insert_text: "/brand-guidelines",
        replace_start: 0,
        replace_end: 17
      },
      {
        kind: "command",
        label: "/build-things",
        detail: "English detail",
        insert_text: "/build-things",
        replace_start: 0,
        replace_end: 13
      },
      {
        kind: "command",
        label: "/canvas-design",
        detail: "English detail",
        insert_text: "/canvas-design",
        replace_start: 0,
        replace_end: 14
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => {
      const map: Record<string, string> = {
        "conversation.composer.suggestion.command.antfu": "应用 Anthony Fu 风格的 JavaScript/TypeScript 工具与约定。",
        "conversation.composer.suggestion.command.brandGuidelines": "应用 Anthropic 官方品牌色与字体规范。",
        "conversation.composer.suggestion.command.buildThings": "生成并打开 Codex Super Bowl 周边兑换链接。",
        "conversation.composer.suggestion.command.canvasDesign": "创建高质量视觉设计（PNG/PDF）。"
      };
      return map[key] ?? key;
    });

    expect(localized.map((item) => item.detail)).toEqual([
      "应用 Anthony Fu 风格的 JavaScript/TypeScript 工具与约定。",
      "应用 Anthropic 官方品牌色与字体规范。",
      "生成并打开 Codex Super Bowl 周边兑换链接。",
      "创建高质量视觉设计（PNG/PDF）。"
    ]);
  });

  it("localizes known skill-set commands using skill key namespace", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "command",
        label: "/doc",
        detail: "English detail",
        insert_text: "/doc",
        replace_start: 0,
        replace_end: 4
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => {
      if (key === "conversation.composer.suggestion.command.skill.doc") {
        return "处理 .docx 文档并尽量保持版式与格式一致。";
      }
      return key;
    });

    expect(localized[0]?.detail).toBe("处理 .docx 文档并尽量保持版式与格式一致。");
  });

  it("shows model name as label and keeps canonical @model:id in meta", () => {
    const suggestions: ComposerSuggestion[] = [
      {
        kind: "resource",
        label: "@model:rc_model_1",
        detail: "MiniMax-M2.5",
        insert_text: "@model:rc_model_1",
        replace_start: 0,
        replace_end: 7
      }
    ];

    const localized = localizeComposerSuggestionDetails(suggestions, (key) => key);
    expect(localized[0]?.label).toBe("@model:MiniMax-M2.5");
    expect(localized[0]?.detail).toBe("@model:rc_model_1");
    expect(localized[0]?.insert_text).toBe("@model:rc_model_1");
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
