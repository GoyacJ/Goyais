import { describe, expect, it } from "vitest";

import { formatTokenCompact, formatTokenUsageWithThreshold } from "@/shared/utils/tokenDisplay";

describe("tokenDisplay", () => {
  it("formats compact token values using decimal K/M", () => {
    expect(formatTokenCompact(0)).toBe("0");
    expect(formatTokenCompact(987)).toBe("987");
    expect(formatTokenCompact(1000)).toBe("1K");
    expect(formatTokenCompact(15603)).toBe("15.6K");
    expect(formatTokenCompact(1_000_000)).toBe("1M");
    expect(formatTokenCompact(2_340_000)).toBe("2.3M");
  });

  it("normalizes invalid values to zero", () => {
    expect(formatTokenCompact(-1)).toBe("0");
    expect(formatTokenCompact(Number.NaN)).toBe("0");
    expect(formatTokenCompact(Number.POSITIVE_INFINITY)).toBe("0");
    expect(formatTokenCompact("123")).toBe("0");
  });

  it("formats token usage with threshold and infinity fallback", () => {
    expect(formatTokenUsageWithThreshold(15603, 200000)).toBe("15.6K / 200K");
    expect(formatTokenUsageWithThreshold(15603, undefined)).toBe("15.6K / ∞");
    expect(formatTokenUsageWithThreshold(15603, 0)).toBe("15.6K / ∞");
  });
});
