import { Globe2, HardDrive, Plus, Server } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { bootstrapAdmin, getBootstrapStatus, getNavigation, listWorkspaces, login, me } from "@/api/hubClient";
import { loadToken, storeToken } from "@/api/secretStoreClient";
import { ensureLocalHubContext } from "@/api/sessionDataSource";
import { RemoteLoginDialog } from "@/components/domain/workspace/RemoteLoginDialog";
import { SetupAdminDialog } from "@/components/domain/workspace/SetupAdminDialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/components/ui/toast";
import { ApiError } from "@/lib/api-error";
import {
  selectCurrentProfile,
  useWorkspaceStore,
  workspaceKey,
  type WorkspaceProfile
} from "@/stores/workspaceStore";

interface WorkspaceSwitcherProps {
  collapsed: boolean;
}

export function WorkspaceSwitcher({ collapsed }: WorkspaceSwitcherProps) {
  const { t } = useTranslation();
  const { addToast } = useToast();
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const profiles = useWorkspaceStore((state) => state.profiles);
  const remoteWorkspacesByProfileId = useWorkspaceStore((state) => state.remoteWorkspacesByProfileId);
  const remoteNavigationByWorkspaceKey = useWorkspaceStore((state) => state.remoteNavigationByWorkspaceKey);

  const setCurrentProfile = useWorkspaceStore((state) => state.setCurrentProfile);
  const upsertRemoteProfile = useWorkspaceStore((state) => state.upsertRemoteProfile);
  const setRemoteWorkspaces = useWorkspaceStore((state) => state.setRemoteWorkspaces);
  const setRemoteSelectedWorkspace = useWorkspaceStore((state) => state.setRemoteSelectedWorkspace);
  const setRemoteNavigation = useWorkspaceStore((state) => state.setRemoteNavigation);
  const setRemoteNavigationLoading = useWorkspaceStore((state) => state.setRemoteNavigationLoading);
  const setRemoteUser = useWorkspaceStore((state) => state.setRemoteUser);

  const [loginOpen, setLoginOpen] = useState(false);
  const [setupOpen, setSetupOpen] = useState(false);
  const [pendingSetupServerUrl, setPendingSetupServerUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string>();

  const localProfiles = useMemo(() => profiles.filter((profile) => profile.kind === "local"), [profiles]);
  const remoteProfiles = useMemo(() => profiles.filter((profile) => profile.kind === "remote"), [profiles]);

  const currentLabel = useMemo(() => {
    if (!currentProfile) {
      return t("workspace.unknown");
    }

    if (currentProfile.kind === "local") {
      return currentProfile.name;
    }

    const selectedWorkspaceId = currentProfile.remote?.selectedWorkspaceId;
    const selectedWorkspace = selectedWorkspaceId
      ? remoteWorkspacesByProfileId[currentProfile.id]?.find((workspace) => workspace.workspace_id === selectedWorkspaceId)
      : undefined;

    return selectedWorkspace ? `${currentProfile.name} / ${selectedWorkspace.name}` : currentProfile.name;
  }, [currentProfile, remoteWorkspacesByProfileId, t]);

  const isUnauthorizedError = useCallback((error: unknown): boolean => {
    return error instanceof ApiError && error.status === 401;
  }, []);

  const hydrateRemoteProfile = async (
    profileId: string,
    profile: WorkspaceProfile,
    token: string,
    preferredWorkspaceId?: string
  ) => {
    const serverUrl = profile.remote?.serverUrl;
    if (!serverUrl) {
      return;
    }

    const workspacesResponse = await listWorkspaces(serverUrl, token);
    setRemoteWorkspaces(profileId, workspacesResponse.workspaces);

    const selectedWorkspaceId = preferredWorkspaceId
      ? preferredWorkspaceId
      : profile.remote?.selectedWorkspaceId || workspacesResponse.workspaces[0]?.workspace_id;

    if (!selectedWorkspaceId) {
      return;
    }

    setRemoteSelectedWorkspace(profileId, selectedWorkspaceId);
    const key = workspaceKey(profileId, selectedWorkspaceId);
    if (!remoteNavigationByWorkspaceKey[key]) {
      setRemoteNavigationLoading(profileId, selectedWorkspaceId, true);
      const navigation = await getNavigation(serverUrl, token, selectedWorkspaceId);
      setRemoteNavigation(profileId, selectedWorkspaceId, navigation);
    }

    setCurrentProfile(profileId);
  };

  const handleLoginSubmit = async (payload: { serverUrl: string; email: string; password: string }) => {
    setLoading(true);
    setErrorMessage(undefined);

    try {
      const status = await getBootstrapStatus(payload.serverUrl);
      if (status.setup_mode) {
        setPendingSetupServerUrl(payload.serverUrl);
        setSetupOpen(true);
        setLoginOpen(false);
        return;
      }

      const loginResponse = await login(payload.serverUrl, {
        email: payload.email,
        password: payload.password
      });

      const profileId = upsertRemoteProfile({
        serverUrl: payload.serverUrl,
        name: payload.serverUrl
      });

      await storeToken(profileId, loginResponse.token);
      const mePayload = await me(payload.serverUrl, loginResponse.token);
      setRemoteUser(profileId, mePayload.user);

      const profile = useWorkspaceStore.getState().profiles.find((item) => item.id === profileId);
      if (profile) {
        await hydrateRemoteProfile(profileId, profile, loginResponse.token);
      }

      setLoginOpen(false);
    } catch (error) {
      const message = error instanceof ApiError ? error.message : (error as Error).message;
      setErrorMessage(message);
    } finally {
      setLoading(false);
    }
  };

  const handleSetupSubmit = async (payload: {
    bootstrapToken: string;
    email: string;
    password: string;
    displayName: string;
  }) => {
    if (!pendingSetupServerUrl) {
      return;
    }

    setLoading(true);
    setErrorMessage(undefined);

    try {
      const response = await bootstrapAdmin(pendingSetupServerUrl, {
        bootstrap_token: payload.bootstrapToken,
        email: payload.email,
        password: payload.password,
        display_name: payload.displayName
      });

      const profileId = upsertRemoteProfile({
        serverUrl: pendingSetupServerUrl,
        name: pendingSetupServerUrl
      });

      await storeToken(profileId, response.token);
      const mePayload = await me(pendingSetupServerUrl, response.token);
      setRemoteUser(profileId, mePayload.user);

      const profile = useWorkspaceStore.getState().profiles.find((item) => item.id === profileId);
      if (profile) {
        await hydrateRemoteProfile(
          profileId,
          profile,
          response.token,
          response.workspace?.workspace_id
        );
      }

      setSetupOpen(false);
      setPendingSetupServerUrl("");
    } catch (error) {
      const message = error instanceof ApiError ? error.message : (error as Error).message;
      setErrorMessage(message);
    } finally {
      setLoading(false);
    }
  };

  const verifyCurrentProfile = useCallback(async () => {
    if (!currentProfile) {
      return;
    }

    if (currentProfile.kind === "local") {
      try {
        await ensureLocalHubContext();
        setErrorMessage(undefined);
        setLoginOpen(false);
        setSetupOpen(false);
      } catch (error) {
        addToast({
          title: t("workspace.errorTitle"),
          description: (error as Error).message,
          variant: "error"
        });
      }
      return;
    }

    if (!currentProfile.remote?.selectedWorkspaceId) {
      return;
    }

    try {
      const token = await loadToken(currentProfile.id);
      if (!token) {
        setLoginOpen(true);
        setErrorMessage(t("workspace.tokenMissing"));
        return;
      }

      const mePayload = await me(currentProfile.remote.serverUrl, token);
      setRemoteUser(currentProfile.id, mePayload.user);

      const workspacesResponse = await listWorkspaces(currentProfile.remote.serverUrl, token);
      setRemoteWorkspaces(currentProfile.id, workspacesResponse.workspaces);
      const nextWorkspaceId =
        workspacesResponse.workspaces.find(
          (workspace) => workspace.workspace_id === currentProfile.remote?.selectedWorkspaceId
        )?.workspace_id ?? workspacesResponse.workspaces[0]?.workspace_id;

      if (!nextWorkspaceId) {
        return;
      }

      if (nextWorkspaceId !== currentProfile.remote.selectedWorkspaceId) {
        setRemoteSelectedWorkspace(currentProfile.id, nextWorkspaceId);
      }

      const key = workspaceKey(currentProfile.id, nextWorkspaceId);
      if (!remoteNavigationByWorkspaceKey[key]) {
        setRemoteNavigationLoading(currentProfile.id, nextWorkspaceId, true);
        const navigation = await getNavigation(currentProfile.remote.serverUrl, token, nextWorkspaceId);
        setRemoteNavigation(currentProfile.id, nextWorkspaceId, navigation);
      }
      setErrorMessage(undefined);
    } catch (error) {
      if (isUnauthorizedError(error)) {
        setLoginOpen(true);
      }
      addToast({
        title: t("workspace.errorTitle"),
        description: (error as Error).message,
        variant: "error"
      });
    }
  }, [
    addToast,
    currentProfile,
    isUnauthorizedError,
    remoteNavigationByWorkspaceKey,
    setRemoteNavigation,
    setRemoteNavigationLoading,
    setRemoteSelectedWorkspace,
    setRemoteUser,
    setRemoteWorkspaces,
    t
  ]);

  useEffect(() => {
    void verifyCurrentProfile();
  }, [verifyCurrentProfile]);

  const handleSelectLocalWorkspace = async (profileId: string) => {
    try {
      await ensureLocalHubContext();
      setErrorMessage(undefined);
      setCurrentProfile(profileId);
    } catch (error) {
      addToast({
        title: t("workspace.errorTitle"),
        description: (error as Error).message,
        variant: "error"
      });
    }
  };

  const handleSelectRemoteWorkspace = async (profile: WorkspaceProfile, workspaceId: string) => {
    if (profile.kind !== "remote" || !profile.remote) {
      return;
    }

    try {
      const token = await loadToken(profile.id);
      if (!token) {
        setLoginOpen(true);
        setErrorMessage(t("workspace.tokenMissing"));
        return;
      }

      const mePayload = await me(profile.remote.serverUrl, token);
      const workspacesResponse = await listWorkspaces(profile.remote.serverUrl, token);
      const nextWorkspaceId =
        workspacesResponse.workspaces.find((workspace) => workspace.workspace_id === workspaceId)?.workspace_id
        ?? workspacesResponse.workspaces[0]?.workspace_id;
      if (!nextWorkspaceId) {
        throw new Error(t("workspace.noWorkspace"));
      }

      const key = workspaceKey(profile.id, nextWorkspaceId);
      let navigation = remoteNavigationByWorkspaceKey[key];
      if (!navigation) {
        setRemoteNavigationLoading(profile.id, nextWorkspaceId, true);
        navigation = await getNavigation(profile.remote.serverUrl, token, nextWorkspaceId);
      }

      setRemoteUser(profile.id, mePayload.user);
      setRemoteWorkspaces(profile.id, workspacesResponse.workspaces);
      setRemoteSelectedWorkspace(profile.id, nextWorkspaceId);
      if (navigation) {
        setRemoteNavigation(profile.id, nextWorkspaceId, navigation);
      }
      setCurrentProfile(profile.id);
      setErrorMessage(undefined);
    } catch (error) {
      if (isUnauthorizedError(error)) {
        setLoginOpen(true);
      }
      addToast({
        title: t("workspace.errorTitle"),
        description: (error as Error).message,
        variant: "error"
      });
    }
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button size={collapsed ? "icon" : "sm"} variant="outline" className={collapsed ? "w-8" : "w-full justify-between"}>
            <span className="flex items-center gap-2 overflow-hidden">
              <Globe2 className="h-4 w-4 shrink-0" />
              {!collapsed ? <span className="truncate text-small">{currentLabel}</span> : null}
            </span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-72">
          <DropdownMenuLabel>{t("workspace.localGroup")}</DropdownMenuLabel>
          {localProfiles.map((profile) => (
            <DropdownMenuItem
              key={profile.id}
              onClick={() => {
                setErrorMessage(undefined);
                void handleSelectLocalWorkspace(profile.id);
              }}
            >
              <HardDrive className="mr-2 h-4 w-4" />
              <span>{profile.name}</span>
            </DropdownMenuItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{t("workspace.remoteGroup")}</DropdownMenuLabel>
          {remoteProfiles.length === 0 ? (
            <DropdownMenuItem disabled>{t("workspace.noRemote")}</DropdownMenuItem>
          ) : (
            remoteProfiles.map((profile) => {
              const workspaces = remoteWorkspacesByProfileId[profile.id] ?? [];
              return (
                <DropdownMenuSub key={profile.id}>
                  <DropdownMenuSubTrigger>
                    <Server className="mr-2 h-4 w-4" />
                    <span className="truncate">{profile.name}</span>
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent className="w-64">
                    {workspaces.length === 0 ? (
                      <DropdownMenuItem disabled>{t("workspace.noWorkspace")}</DropdownMenuItem>
                    ) : (
                      workspaces.map((workspace) => (
                        <DropdownMenuItem
                          key={workspace.workspace_id}
                          onClick={() => void handleSelectRemoteWorkspace(profile, workspace.workspace_id)}
                        >
                          <span className="truncate">{workspace.name}</span>
                        </DropdownMenuItem>
                      ))
                    )}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              );
            })
          )}

          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => {
              setErrorMessage(undefined);
              setLoginOpen(true);
            }}
          >
            <Plus className="mr-2 h-4 w-4" />
            <span>{t("workspace.addRemote")}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <RemoteLoginDialog
        open={loginOpen}
        loading={loading}
        errorMessage={errorMessage}
        onOpenChange={setLoginOpen}
        onSubmit={handleLoginSubmit}
      />

      <SetupAdminDialog
        open={setupOpen}
        serverUrl={pendingSetupServerUrl}
        loading={loading}
        errorMessage={errorMessage}
        onOpenChange={setSetupOpen}
        onSubmit={handleSetupSubmit}
      />
    </>
  );
}
