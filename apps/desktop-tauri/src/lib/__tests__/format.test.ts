import { describe, expect, it } from "vitest";

import { formatTimeByLocale } from "@/lib/format";

describe("formatTimeByLocale", () => {
  it("formats with locale-specific output", () => {
    const ts = "2026-02-20T10:30:00.000Z";
    const zh = formatTimeByLocale(ts, "zh-CN");
    const en = formatTimeByLocale(ts, "en-US");

    expect(zh).not.toBe(en);
    expect(zh.length).toBeGreaterThan(0);
    expect(en.length).toBeGreaterThan(0);
  });

  it("returns empty string for invalid timestamp", () => {
    expect(formatTimeByLocale("bad-ts", "zh-CN")).toBe("");
  });
});
