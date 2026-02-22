import { describe, expect, it } from "vitest";

import ResourceView from "@/modules/resource/views/ResourceView.vue";

describe("resource view", () => {
  it("is defined", () => {
    expect(ResourceView).toBeTruthy();
  });
});
