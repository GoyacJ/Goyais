import { beforeEach, describe, expect, it } from "vitest";

import {
  resetWorkspaceStore,
  setCurrentWorkspace,
  setWorkspaces,
  workspaceStore
} from "@/shared/stores/workspaceStore";
import type { Workspace } from "@/shared/types/api";

const localWorkspace: Workspace = {
  id: "ws_local",
  name: "Local Workspace",
  mode: "local",
  hub_url: null,
  is_default_local: true,
  created_at: "2026-02-23T00:00:00Z",
  login_disabled: true,
  auth_mode: "disabled"
};

const remoteCompany: Workspace = {
  id: "ws_remote_company",
  name: "公司工作区",
  mode: "remote",
  hub_url: "https://hub.company.local",
  is_default_local: false,
  created_at: "2026-02-23T00:00:00Z",
  login_disabled: false,
  auth_mode: "password_or_token"
};

const remoteApple: Workspace = {
  id: "ws_remote_apple",
  name: "苹果项目工作区",
  mode: "remote",
  hub_url: "https://hub.apple.local",
  is_default_local: false,
  created_at: "2026-02-23T01:00:00Z",
  login_disabled: false,
  auth_mode: "password_or_token"
};

describe("workspaceStore", () => {
  beforeEach(() => {
    resetWorkspaceStore();
  });

  it("只保留本地工作区时，下拉源仅包含本地", () => {
    setWorkspaces([localWorkspace]);
    expect(workspaceStore.workspaces.map((workspace) => workspace.id)).toEqual(["ws_local"]);
  });

  it("远程工作区按最近使用排序", () => {
    setWorkspaces([localWorkspace, remoteCompany, remoteApple]);

    setCurrentWorkspace("ws_remote_apple");
    expect(workspaceStore.workspaces.map((workspace) => workspace.id)).toEqual([
      "ws_local",
      "ws_remote_apple",
      "ws_remote_company"
    ]);

    setCurrentWorkspace("ws_remote_company");
    expect(workspaceStore.workspaces.map((workspace) => workspace.id)).toEqual([
      "ws_local",
      "ws_remote_company",
      "ws_remote_apple"
    ]);
  });

  it("后端未返回本地时自动注入本地工作区", () => {
    setWorkspaces([remoteCompany]);
    expect(workspaceStore.workspaces[0]?.id).toBe("ws_local");
    expect(workspaceStore.workspaces[1]?.id).toBe("ws_remote_company");
  });
});
