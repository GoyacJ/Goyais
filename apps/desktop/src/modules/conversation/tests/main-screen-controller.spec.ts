import { describe, expect, it } from "vitest";

import { MAIN_INSPECTOR_TABS } from "@/modules/conversation/views/useMainScreenController";

describe("main screen controller", () => {
  it("exposes inspector tabs without files", () => {
    const keys = MAIN_INSPECTOR_TABS.map((item) => item.key);
    expect(keys).toEqual(["diff", "run", "trace", "risk"]);
    expect(keys).not.toContain("files");
  });
});
