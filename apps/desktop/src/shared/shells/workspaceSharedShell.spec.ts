import { describe, expect, it } from "vitest";

import { resolveWorkspaceSharedShell } from "@/shared/shells/workspaceSharedShell";
import type { WorkspaceMode } from "@/shared/types/api";

describe("workspace shared shell", () => {
  it("uses account shell in remote mode and settings shell in local mode", () => {
    expect(resolveWorkspaceSharedShell("remote")).toBe("account");
    expect(resolveWorkspaceSharedShell("local")).toBe("settings");
  });

  it("keeps return type stable for workspace modes", () => {
    const modes: WorkspaceMode[] = ["local", "remote"];
    const shells = modes.map((mode) => resolveWorkspaceSharedShell(mode));
    expect(shells).toEqual(["settings", "account"]);
  });
});
