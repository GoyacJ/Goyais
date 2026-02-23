import type { WorkspaceMode } from "@/shared/types/api";

export type WorkspaceSharedShell = "account" | "settings";

export function resolveWorkspaceSharedShell(mode: WorkspaceMode): WorkspaceSharedShell {
  return mode === "remote" ? "account" : "settings";
}
