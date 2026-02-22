import { describe, expect, it } from "vitest";

import WorkspaceView from "@/modules/workspace/views/WorkspaceView.vue";

describe("workspace view", () => {
  it("is defined", () => {
    expect(WorkspaceView).toBeTruthy();
  });
});
