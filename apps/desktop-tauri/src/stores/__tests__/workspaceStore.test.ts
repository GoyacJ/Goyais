import { beforeEach, describe, expect, it } from "vitest";

import {
  selectCurrentPermissions,
  useWorkspaceStore,
  workspaceKey,
  type WorkspaceProfile
} from "@/stores/workspaceStore";

function makeLocalProfile(): WorkspaceProfile {
  return {
    id: "local-default",
    kind: "local",
    name: "Local Workspace",
    local: {
      rootPath: "/Users/goya/Repo/Git/Goyais"
    },
    lastUsedAt: new Date().toISOString()
  };
}

describe("workspaceStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useWorkspaceStore.setState({
      profiles: [makeLocalProfile()],
      currentProfileId: "local-default",
      remoteWorkspacesByProfileId: {},
      remoteNavigationByWorkspaceKey: {},
      remoteNavigationLoadingByWorkspaceKey: {},
      remoteUsersByProfileId: {}
    });
  });

  it("creates remote profile with tokenRef = profileId", () => {
    const profileId = useWorkspaceStore.getState().upsertRemoteProfile({
      serverUrl: "http://127.0.0.1:8787"
    });

    const profile = useWorkspaceStore.getState().profiles.find((item) => item.id === profileId);
    expect(profile?.kind).toBe("remote");
    expect(profile?.remote?.tokenRef).toBe(profileId);
  });

  it("returns workspace-scoped permissions for current remote workspace", () => {
    const profileId = useWorkspaceStore.getState().upsertRemoteProfile({
      serverUrl: "http://127.0.0.1:8787"
    });

    useWorkspaceStore.getState().setCurrentProfile(profileId);
    useWorkspaceStore.getState().setRemoteSelectedWorkspace(profileId, "ws-1");
    useWorkspaceStore.getState().setRemoteNavigation(profileId, "ws-1", {
      workspace_id: "ws-1",
      menus: [],
      permissions: ["project:read", "run:create"],
      feature_flags: {}
    });

    const state = useWorkspaceStore.getState();
    expect(state.remoteNavigationByWorkspaceKey[workspaceKey(profileId, "ws-1")]).toBeDefined();
    expect(selectCurrentPermissions(state)).toEqual(["project:read", "run:create"]);
  });

  it("returns stable permissions reference for local workspace", () => {
    const state = useWorkspaceStore.getState();

    const first = selectCurrentPermissions(state);
    const second = selectCurrentPermissions(state);

    expect(first).toBe(second);
  });

  it("returns stable empty permissions reference for remote workspace without navigation", () => {
    const profileId = useWorkspaceStore.getState().upsertRemoteProfile({
      serverUrl: "http://127.0.0.1:8787"
    });
    useWorkspaceStore.getState().setCurrentProfile(profileId);

    const state = useWorkspaceStore.getState();
    const first = selectCurrentPermissions(state);
    const second = selectCurrentPermissions(state);

    expect(first).toBe(second);
    expect(first).toEqual([]);
  });
});
