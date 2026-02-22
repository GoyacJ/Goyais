import { describe, expect, it } from "vitest";

import AdminView from "@/modules/admin/views/AdminView.vue";

describe("admin view", () => {
  it("is defined", () => {
    expect(AdminView).toBeTruthy();
  });
});
