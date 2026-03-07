import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";

const runtimeMocks = vi.hoisted(() => ({
  isRuntimeCapabilitySupported: vi.fn(() => true)
}));

vi.mock("@/shared/runtime", () => ({
  isRuntimeCapabilitySupported: runtimeMocks.isRuntimeCapabilitySupported
}));

import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

describe("WorkspaceSwitcherCard", () => {
  afterEach(() => {
    runtimeMocks.isRuntimeCapabilitySupported.mockReset();
    runtimeMocks.isRuntimeCapabilitySupported.mockReturnValue(true);
  });

  it("renders traffic-light glyphs for mac-style controls", () => {
    const wrapper = mount(WorkspaceSwitcherCard, {
      props: {
        workspaces: [
          {
            id: "ws_local",
            name: "Local",
            mode: "local"
          }
        ] as never[],
        currentWorkspaceId: "ws_local"
      }
    });

    const glyphs = wrapper.findAll(".dot-glyph");
    expect(glyphs).toHaveLength(3);
    for (const glyph of glyphs) {
      expect(glyph.classes()).toContain("group-hover:opacity-100");
      expect(glyph.classes()).toContain("group-focus-within:opacity-100");
    }
  });
});
