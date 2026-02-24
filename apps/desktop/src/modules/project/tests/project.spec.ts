import { describe, expect, it } from "vitest";

import ProjectView from "@/modules/project/views/ProjectView.vue";

describe("project view", () => {
  it("is defined", () => {
    expect(ProjectView).toBeTruthy();
  });
});
