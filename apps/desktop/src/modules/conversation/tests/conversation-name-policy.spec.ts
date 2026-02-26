import { describe, expect, it } from "vitest";

import { buildNameFromFirstMessage, isDefaultConversationName } from "@/modules/conversation/views/conversationNamePolicy";

describe("conversation name policy", () => {
  it("recognizes default conversation names in Chinese and English", () => {
    expect(isDefaultConversationName("新对话")).toBe(true);
    expect(isDefaultConversationName("新对话 1")).toBe(true);
    expect(isDefaultConversationName("  新对话   23  ")).toBe(true);
    expect(isDefaultConversationName("Conversation")).toBe(true);
    expect(isDefaultConversationName("Conversation 3")).toBe(true);
  });

  it("does not match custom or non-number-suffixed names", () => {
    expect(isDefaultConversationName("我的会话")).toBe(false);
    expect(isDefaultConversationName("Conversation x")).toBe(false);
    expect(isDefaultConversationName("新对话 alpha")).toBe(false);
  });

  it("normalizes multiline content and truncates to 10 unicode characters by default", () => {
    const name = buildNameFromFirstMessage("  第一行\n第二行   第三行  ");
    expect(name).toBe("第一行 第二行 第三");
    expect(Array.from(name)).toHaveLength(10);
  });

  it("truncates mixed Chinese and ASCII by unicode character count", () => {
    expect(buildNameFromFirstMessage("hello世界12345")).toBe("hello世界123");
  });

  it("returns empty string when content is blank after normalization", () => {
    expect(buildNameFromFirstMessage(" \n\t  ")).toBe("");
  });

  it("supports custom max chars", () => {
    expect(buildNameFromFirstMessage("abcdef", 3)).toBe("abc");
  });
});
