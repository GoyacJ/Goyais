import { describe, expect, it } from "vitest";

import {
  isSameConversationResponseTarget,
  MAIN_INSPECTOR_TABS,
  shouldApplyRunTaskDetailResponse
} from "@/modules/conversation/views/useMainScreenController";

describe("main screen controller", () => {
  it("exposes inspector tabs without files", () => {
    const keys = MAIN_INSPECTOR_TABS.map((item) => item.key);
    expect(keys).toEqual(["diff", "run", "trace", "risk"]);
    expect(keys).not.toContain("files");
  });

  it("matches conversation response target by normalized id", () => {
    expect(isSameConversationResponseTarget(" conv_1 ", "conv_1")).toBe(true);
    expect(isSameConversationResponseTarget("conv_1", "conv_2")).toBe(false);
    expect(isSameConversationResponseTarget("", "conv_1")).toBe(false);
  });

  it("guards run task detail response by conversation and selected task", () => {
    expect(shouldApplyRunTaskDetailResponse("conv_1", "conv_1", "task_1", "task_1")).toBe(true);
    expect(shouldApplyRunTaskDetailResponse("conv_1", "conv_2", "task_1", "task_1")).toBe(false);
    expect(shouldApplyRunTaskDetailResponse("conv_1", "conv_1", "task_1", "task_2")).toBe(false);
    expect(shouldApplyRunTaskDetailResponse("conv_1", "conv_1", "", "task_1")).toBe(false);
  });
});
