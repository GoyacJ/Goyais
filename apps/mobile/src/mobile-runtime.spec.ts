import { describe, expect, it } from "vitest";

import { resolveRuntimeCapabilities, resolveRuntimeTarget } from "@/shared/runtime";

describe("mobile runtime wiring", () => {
  it("resolves mobile target from env marker", () => {
    expect(resolveRuntimeTarget("mobile")).toBe("mobile");
  });

  it("keeps sidecar disabled on mobile", () => {
    const capabilities = resolveRuntimeCapabilities("mobile");
    expect(capabilities.supportsSidecar).toBe(false);
    expect(capabilities.supportsLocalWorkspace).toBe(false);
  });
});
