import { describe, expect, it } from "vitest";

import { canManageModelConfigs } from "@/pages/ModelConfigsPage";
import { canWriteProjects } from "@/pages/ProjectsPage";

describe("remote permissions gating", () => {
  it("requires project:write for remote project mutations", () => {
    expect(canWriteProjects("remote", ["project:read"])).toBe(false);
    expect(canWriteProjects("remote", ["project:read", "project:write"])).toBe(true);
  });

  it("requires modelconfig:manage for remote model config mutations", () => {
    expect(canManageModelConfigs("remote", ["modelconfig:read"])).toBe(false);
    expect(canManageModelConfigs("remote", ["modelconfig:read", "modelconfig:manage"])).toBe(true);
  });

  it("keeps local mode writable", () => {
    expect(canWriteProjects("local", [])).toBe(true);
    expect(canManageModelConfigs("local", [])).toBe(true);
  });
});
