import { ReactNode } from "react";

import { LoadingState } from "@/components/domain/feedback/LoadingState";
import { NoPermissionPage } from "@/pages/NoPermissionPage";
import {
  selectCurrentPermissions,
  selectCurrentProfile,
  selectCurrentWorkspaceKind,
  useWorkspaceStore,
  workspaceKey
} from "@/stores/workspaceStore";

interface PermissionGateProps {
  routePath: string;
  children: ReactNode;
}

const ROUTE_PERMISSION_MAP: Record<string, string | undefined> = {
  "/": "run:create",
  "/run": "run:create",
  "/projects": "project:read",
  "/models": "modelconfig:read",
  "/replay": "run:read",
  "/settings": "workspace:read"
};

export function PermissionGate({ routePath, children }: PermissionGateProps) {
  const currentKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const remoteNavigationByWorkspaceKey = useWorkspaceStore((state) => state.remoteNavigationByWorkspaceKey);

  if (currentKind === "local") {
    return <>{children}</>;
  }

  const workspaceId = currentProfile?.remote?.selectedWorkspaceId;
  if (!currentProfile || !workspaceId) {
    return <LoadingState label="Loading workspace..." />;
  }

  const navKey = workspaceKey(currentProfile.id, workspaceId);
  const navigationLoaded = Boolean(remoteNavigationByWorkspaceKey[navKey]);
  if (!navigationLoaded) {
    return <LoadingState label="Loading navigation..." />;
  }

  const requiredPermission = ROUTE_PERMISSION_MAP[routePath];
  if (requiredPermission && !permissions.includes(requiredPermission)) {
    return <NoPermissionPage />;
  }

  return <>{children}</>;
}
