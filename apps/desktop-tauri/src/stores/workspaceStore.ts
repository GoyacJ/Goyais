import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export interface WorkspaceProfile {
  id: string;
  kind: "local" | "remote";
  name: string;
  local?: {
    rootPath: string;
  };
  remote?: {
    serverUrl: string;
    tokenRef: string;
    selectedWorkspaceId?: string;
  };
  lastUsedAt: string;
}

export interface RemoteWorkspaceSummary {
  workspace_id: string;
  name: string;
  slug: string;
  role_name: string;
}

export interface RemoteNavigationMenu {
  menu_id: string;
  route: string | null;
  icon_key: string | null;
  i18n_key: string;
  children: RemoteNavigationMenu[];
}

export interface RemoteNavigationBucket {
  workspace_id: string;
  menus: RemoteNavigationMenu[];
  permissions: string[];
  feature_flags: Record<string, boolean>;
}

export interface RemoteUserProfile {
  user_id: string;
  email: string;
  display_name: string;
}

interface WorkspaceStoreState {
  profiles: WorkspaceProfile[];
  currentProfileId: string;
  remoteWorkspacesByProfileId: Record<string, RemoteWorkspaceSummary[]>;
  remoteNavigationByWorkspaceKey: Record<string, RemoteNavigationBucket>;
  remoteNavigationLoadingByWorkspaceKey: Record<string, boolean>;
  remoteUsersByProfileId: Record<string, RemoteUserProfile>;
  ensureLocalProfile: () => void;
  upsertRemoteProfile: (params: { serverUrl: string; name?: string }) => string;
  setCurrentProfile: (profileId: string) => void;
  touchProfile: (profileId: string) => void;
  setRemoteWorkspaces: (profileId: string, workspaces: RemoteWorkspaceSummary[]) => void;
  setRemoteSelectedWorkspace: (profileId: string, workspaceId: string) => void;
  setRemoteNavigation: (profileId: string, workspaceId: string, bucket: RemoteNavigationBucket) => void;
  setRemoteNavigationLoading: (profileId: string, workspaceId: string, loading: boolean) => void;
  setRemoteUser: (profileId: string, user: RemoteUserProfile) => void;
  clearRemoteAuth: (profileId: string) => void;
}

const DEFAULT_LOCAL_PROFILE_ID = "local-default";
const DEFAULT_LOCAL_ROOT = "/Users/goya/Repo/Git/Goyais";
const LOCAL_PERMISSIONS: string[] = [
  "workspace:read",
  "workspace:manage",
  "modelconfig:read",
  "modelconfig:manage",
  "project:read",
  "project:write",
  "run:create",
  "run:read",
  "confirm:write",
  "audit:read"
];
const EMPTY_PERMISSIONS: string[] = [];

function nowIso(): string {
  return new Date().toISOString();
}

function makeWorkspaceKey(profileId: string, workspaceId: string): string {
  return `${profileId}:${workspaceId}`;
}

function defaultLocalProfile(): WorkspaceProfile {
  return {
    id: DEFAULT_LOCAL_PROFILE_ID,
    kind: "local",
    name: "Local Workspace",
    local: {
      rootPath: DEFAULT_LOCAL_ROOT
    },
    lastUsedAt: nowIso()
  };
}

export const useWorkspaceStore = create<WorkspaceStoreState>()(
  persist(
    (set, get) => ({
      profiles: [defaultLocalProfile()],
      currentProfileId: DEFAULT_LOCAL_PROFILE_ID,
      remoteWorkspacesByProfileId: {},
      remoteNavigationByWorkspaceKey: {},
      remoteNavigationLoadingByWorkspaceKey: {},
      remoteUsersByProfileId: {},
      ensureLocalProfile: () => {
        const hasLocal = get().profiles.some((profile) => profile.kind === "local");
        if (hasLocal) {
          return;
        }

        set((state) => ({
          profiles: [defaultLocalProfile(), ...state.profiles],
          currentProfileId: state.currentProfileId || DEFAULT_LOCAL_PROFILE_ID
        }));
      },
      upsertRemoteProfile: ({ serverUrl, name }) => {
        const normalizedUrl = serverUrl.trim().replace(/\/+$/, "");
        const existing = get().profiles.find(
          (profile) => profile.kind === "remote" && profile.remote?.serverUrl === normalizedUrl
        );

        if (existing) {
          set((state) => ({
            profiles: state.profiles.map((profile) =>
              profile.id === existing.id
                ? {
                    ...profile,
                    name: name?.trim() || existing.name,
                    lastUsedAt: nowIso()
                  }
                : profile
            )
          }));
          return existing.id;
        }

        const profileId = crypto.randomUUID();
        const nextProfile: WorkspaceProfile = {
          id: profileId,
          kind: "remote",
          name: name?.trim() || normalizedUrl,
          remote: {
            serverUrl: normalizedUrl,
            tokenRef: profileId
          },
          lastUsedAt: nowIso()
        };

        set((state) => ({
          profiles: [...state.profiles, nextProfile]
        }));

        return profileId;
      },
      setCurrentProfile: (profileId) => {
        set((state) => {
          const profile = state.profiles.find((item) => item.id === profileId);
          if (!profile) {
            return state;
          }

          return {
            currentProfileId: profileId,
            profiles: state.profiles.map((item) =>
              item.id === profileId
                ? {
                    ...item,
                    lastUsedAt: nowIso()
                  }
                : item
            )
          };
        });
      },
      touchProfile: (profileId) => {
        set((state) => ({
          profiles: state.profiles.map((item) =>
            item.id === profileId
              ? {
                  ...item,
                  lastUsedAt: nowIso()
                }
              : item
          )
        }));
      },
      setRemoteWorkspaces: (profileId, workspaces) => {
        set((state) => ({
          remoteWorkspacesByProfileId: {
            ...state.remoteWorkspacesByProfileId,
            [profileId]: workspaces
          }
        }));
      },
      setRemoteSelectedWorkspace: (profileId, workspaceId) => {
        set((state) => ({
          profiles: state.profiles.map((profile) => {
            if (profile.id !== profileId || profile.kind !== "remote" || !profile.remote) {
              return profile;
            }

            return {
              ...profile,
              remote: {
                ...profile.remote,
                selectedWorkspaceId: workspaceId
              },
              lastUsedAt: nowIso()
            };
          })
        }));
      },
      setRemoteNavigation: (profileId, workspaceId, bucket) => {
        const key = makeWorkspaceKey(profileId, workspaceId);
        set((state) => ({
          remoteNavigationByWorkspaceKey: {
            ...state.remoteNavigationByWorkspaceKey,
            [key]: bucket
          },
          remoteNavigationLoadingByWorkspaceKey: {
            ...state.remoteNavigationLoadingByWorkspaceKey,
            [key]: false
          }
        }));
      },
      setRemoteNavigationLoading: (profileId, workspaceId, loading) => {
        const key = makeWorkspaceKey(profileId, workspaceId);
        set((state) => ({
          remoteNavigationLoadingByWorkspaceKey: {
            ...state.remoteNavigationLoadingByWorkspaceKey,
            [key]: loading
          }
        }));
      },
      setRemoteUser: (profileId, user) => {
        set((state) => ({
          remoteUsersByProfileId: {
            ...state.remoteUsersByProfileId,
            [profileId]: user
          }
        }));
      },
      clearRemoteAuth: (profileId) => {
        set((state) => {
          const profile = state.profiles.find((item) => item.id === profileId);
          const selectedWorkspaceId =
            profile?.kind === "remote" ? profile.remote?.selectedWorkspaceId : undefined;
          const nextUsers = { ...state.remoteUsersByProfileId };
          const nextWorkspaces = { ...state.remoteWorkspacesByProfileId };
          const nextNavigation = { ...state.remoteNavigationByWorkspaceKey };
          const nextLoading = { ...state.remoteNavigationLoadingByWorkspaceKey };
          delete nextUsers[profileId];
          delete nextWorkspaces[profileId];
          if (selectedWorkspaceId) {
            const key = makeWorkspaceKey(profileId, selectedWorkspaceId);
            delete nextNavigation[key];
            delete nextLoading[key];
          }

          return {
            profiles: state.profiles.map((item) => {
              if (item.id !== profileId || item.kind !== "remote" || !item.remote) {
                return item;
              }
              return {
                ...item,
                remote: {
                  ...item.remote,
                  selectedWorkspaceId: undefined
                }
              };
            }),
            currentProfileId:
              state.currentProfileId === profileId
                ? state.profiles.find((item) => item.kind === "local")?.id ?? state.currentProfileId
                : state.currentProfileId,
            remoteUsersByProfileId: nextUsers,
            remoteWorkspacesByProfileId: nextWorkspaces,
            remoteNavigationByWorkspaceKey: nextNavigation,
            remoteNavigationLoadingByWorkspaceKey: nextLoading
          };
        });
      }
    }),
    {
      name: "goyais.workspace.registry.v1",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        profiles: state.profiles,
        currentProfileId: state.currentProfileId,
        remoteWorkspacesByProfileId: state.remoteWorkspacesByProfileId,
        remoteNavigationByWorkspaceKey: state.remoteNavigationByWorkspaceKey,
        remoteUsersByProfileId: state.remoteUsersByProfileId
      })
    }
  )
);

export function workspaceKey(profileId: string, workspaceId: string): string {
  return makeWorkspaceKey(profileId, workspaceId);
}

export function selectCurrentProfile(state: WorkspaceStoreState): WorkspaceProfile | undefined {
  return state.profiles.find((profile) => profile.id === state.currentProfileId);
}

export function selectCurrentWorkspaceKind(state: WorkspaceStoreState): "local" | "remote" {
  return selectCurrentProfile(state)?.kind ?? "local";
}

export function selectCurrentRemoteWorkspaceId(state: WorkspaceStoreState): string | undefined {
  const current = selectCurrentProfile(state);
  if (!current || current.kind !== "remote") {
    return undefined;
  }

  return current.remote?.selectedWorkspaceId;
}

export function selectCurrentNavigation(state: WorkspaceStoreState): RemoteNavigationBucket | undefined {
  const current = selectCurrentProfile(state);
  if (!current || current.kind !== "remote") {
    return undefined;
  }

  const workspaceId = current.remote?.selectedWorkspaceId;
  if (!workspaceId) {
    return undefined;
  }

  return state.remoteNavigationByWorkspaceKey[workspaceKey(current.id, workspaceId)];
}

export function selectCurrentPermissions(state: WorkspaceStoreState): string[] {
  if (selectCurrentWorkspaceKind(state) === "local") {
    return LOCAL_PERMISSIONS;
  }

  return selectCurrentNavigation(state)?.permissions ?? EMPTY_PERMISSIONS;
}

useWorkspaceStore.getState().ensureLocalProfile();
