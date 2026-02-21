import { describe, expect, it } from "vitest";

import { CONVERSATION_HEADER_FIELDS, RIGHT_PANEL_SECTION_ORDER } from "@/pages/ConversationPage";

describe("conversation layout ordering", () => {
  it("keeps timeline below diff in execution-only layout", () => {
    expect(RIGHT_PANEL_SECTION_ORDER).toEqual(["diff", "timeline", "tools", "context"]);
    expect(RIGHT_PANEL_SECTION_ORDER.indexOf("diff")).toBeLessThan(RIGHT_PANEL_SECTION_ORDER.indexOf("timeline"));
  });

  it("keeps header fields project/session/branch in fixed order", () => {
    expect(CONVERSATION_HEADER_FIELDS).toEqual(["project", "session", "branch"]);
  });
});
