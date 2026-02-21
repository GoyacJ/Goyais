import { invoke } from "@tauri-apps/api/core";

export async function getGitCurrentBranch(workspacePath: string): Promise<string | null> {
  const branch = await invoke<string | null>("git_current_branch", {
    workspacePath
  });
  return branch ?? null;
}
