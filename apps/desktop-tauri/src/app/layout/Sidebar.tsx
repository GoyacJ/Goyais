import { open } from "@tauri-apps/plugin-dialog";
import type { LucideIcon } from "lucide-react";
import {
  Circle,
  FolderKanban,
  LogOut,
  MessageSquare,
  Plus,
  Settings2,
  Sparkles,
  UserCircle2
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

import { type DataProject, getProjectsClient } from "@/api/dataSource";
import { getRunDataSource } from "@/api/runDataSource";
import { deleteToken } from "@/api/secretStoreClient";
import { WorkspaceSwitcher } from "@/app/layout/WorkspaceSwitcher";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/components/ui/toast";
import { normalizeUnknownError } from "@/lib/api-error";
import { cn } from "@/lib/cn";
import {
  type ConversationSessionSummary,
  useConversationStore
} from "@/stores/conversationStore";
import { useUiStore } from "@/stores/uiStore";
import {
  type RemoteNavigationMenu,
  selectCurrentNavigation,
  selectCurrentProfile,
  selectCurrentWorkspaceKind,
  useWorkspaceStore
} from "@/stores/workspaceStore";

interface SidebarProjectSectionProps {
  collapsed: boolean;
  project: DataProject;
  sessions: ConversationSessionSummary[];
  selectedProjectId?: string;
  selectedSessionId?: string;
  onSelectProject: (projectId: string) => void;
  onSelectSession: (projectId: string, sessionId: string) => void;
  onNewThread: (projectId: string) => void;
}

function SidebarProjectSection({
  collapsed,
  project,
  sessions,
  selectedProjectId,
  selectedSessionId,
  onSelectProject,
  onSelectSession,
  onNewThread
}: SidebarProjectSectionProps) {
  const isActiveProject = selectedProjectId === project.project_id;

  return (
    <section className="space-y-1">
      <div
        className={cn(
          "flex items-center rounded-control px-2 py-1 text-small transition-colors",
          isActiveProject ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/70"
        )}
      >
        <button
          type="button"
          className={cn(
            "flex min-w-0 flex-1 items-center gap-2 text-left",
            collapsed ? "justify-center" : "justify-start"
          )}
          onClick={() => onSelectProject(project.project_id)}
        >
          <FolderKanban className="h-4 w-4 shrink-0" />
          {!collapsed ? <span className="truncate">{project.name}</span> : null}
        </button>
        {!collapsed ? (
          <button
            type="button"
            className="inline-flex h-6 w-6 items-center justify-center rounded-control text-muted-foreground hover:bg-background hover:text-foreground"
            onClick={() => onNewThread(project.project_id)}
            aria-label="new-thread"
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        ) : null}
      </div>

      {!collapsed && isActiveProject ? (
        <div className="ml-5 space-y-1 border-l border-border-subtle pl-2">
          {sessions.map((session) => (
            <button
              key={session.session_id}
              type="button"
              className={cn(
                "flex w-full items-center gap-2 rounded-control px-2 py-1 text-left text-small transition-colors",
                selectedSessionId === session.session_id
                  ? "bg-accent/20 text-accent"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
              onClick={() => onSelectSession(project.project_id, session.session_id)}
            >
              <MessageSquare className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{session.title}</span>
            </button>
          ))}
        </div>
      ) : null}
    </section>
  );
}

interface ExtensionNavItem {
  route: string;
  icon: LucideIcon;
  labelKey: string;
}

const extensionIconMap: Record<string, LucideIcon> = {
  folder: FolderKanban,
  settings: Settings2
};

function flattenRemoteMenus(menus: RemoteNavigationMenu[]): ExtensionNavItem[] {
  const items: ExtensionNavItem[] = [];
  for (const menu of menus) {
    if (menu.route) {
      items.push({
        route: menu.route,
        labelKey: menu.i18n_key,
        icon: menu.icon_key ? extensionIconMap[menu.icon_key] ?? Circle : Circle
      });
    }
    if (menu.children.length > 0) {
      items.push(...flattenRemoteMenus(menu.children));
    }
  }
  return items;
}

function normalizeLocation(location: string): string {
  return location.replace(/\\/g, "/").replace(/\/+$/, "");
}

function deriveProjectName(location: string, fallback: string): string {
  const normalized = normalizeLocation(location);
  const segments = normalized.split("/").filter(Boolean);
  const candidate = segments.at(-1)?.trim();
  return candidate ? candidate : fallback;
}

function toRemoteRootUri(location: string): string {
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(location)) {
    return location;
  }

  const normalized = normalizeLocation(location);
  if (!normalized) {
    return "file:///";
  }

  return normalized.startsWith("/") ? `file://${normalized}` : `file:///${normalized}`;
}

export function Sidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { addToast } = useToast();
  const collapsed = useUiStore((state) => state.sidebarCollapsed);
  const toggleSidebar = useUiStore((state) => state.toggleSidebar);

  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const currentNavigation = useWorkspaceStore(selectCurrentNavigation);
  const remoteUsersByProfileId = useWorkspaceStore((state) => state.remoteUsersByProfileId);
  const clearRemoteAuth = useWorkspaceStore((state) => state.clearRemoteAuth);

  const selectedProjectId = useConversationStore((state) => state.selectedProjectId);
  const selectedSessionId = useConversationStore((state) => state.selectedSessionId);
  const sessionsByProjectId = useConversationStore((state) => state.sessionsByProjectId);
  const setSelectedProject = useConversationStore((state) => state.setSelectedProject);
  const setSelectedSession = useConversationStore((state) => state.setSelectedSession);
  const setSessions = useConversationStore((state) => state.setSessions);
  const upsertSession = useConversationStore((state) => state.upsertSession);

  const projectsClient = useMemo(() => getProjectsClient(currentProfile), [currentProfile]);
  const runDataSource = useMemo(() => getRunDataSource(currentProfile), [currentProfile]);

  const [projects, setProjects] = useState<DataProject[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [creatingProject, setCreatingProject] = useState(false);

  const extensionItems = useMemo(() => flattenRemoteMenus(currentNavigation?.menus ?? []), [currentNavigation]);

  const refreshProjects = useCallback(async () => {
    setLoadingProjects(true);
    try {
      const list = await projectsClient.list();
      setProjects(list);
      if (!selectedProjectId || !list.some((item) => item.project_id === selectedProjectId)) {
        setSelectedProject(list[0]?.project_id);
      }
    } catch (error) {
      addToast({
        title: t("projects.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setLoadingProjects(false);
    }
  }, [addToast, projectsClient, selectedProjectId, setSelectedProject, t]);

  useEffect(() => {
    void refreshProjects();
  }, [refreshProjects]);

  useEffect(() => {
    if (!selectedProjectId) return;

    runDataSource
      .listSessions(selectedProjectId)
      .then((payload) => setSessions(selectedProjectId, payload.sessions))
      .catch(() => {
        // Phase-1 fallback: keep local cache if session API is temporarily unavailable.
      });
  }, [runDataSource, selectedProjectId, setSessions]);

  const onNewThread = async (projectId: string) => {
    try {
      const payload = await runDataSource.createSession({
        project_id: projectId,
        title: t("conversation.newThread")
      });
      upsertSession(payload.session);
      setSelectedSession(projectId, payload.session.session_id);
    } catch {
      const fallback = {
        session_id: crypto.randomUUID(),
        project_id: projectId,
        title: t("conversation.newThread"),
        updated_at: new Date().toISOString()
      };
      upsertSession(fallback);
      setSelectedSession(projectId, fallback.session_id);
    }
  };

  const onCreateProjectFromLocation = useCallback(
    async (rawLocation: string) => {
      const normalizedLocation = rawLocation.trim();
      if (!normalizedLocation) {
        return;
      }

      const location = workspaceKind === "remote" ? toRemoteRootUri(normalizedLocation) : normalizeLocation(normalizedLocation);
      const name = deriveProjectName(normalizedLocation, t("projects.defaultName"));

      setCreatingProject(true);
      try {
        await projectsClient.create({ name, location });
        await refreshProjects();
        addToast({
          title: t("projects.createSuccess"),
          description: name,
          variant: "success"
        });
      } catch (error) {
        addToast({
          title: t("projects.createFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      } finally {
        setCreatingProject(false);
      }
    },
    [addToast, projectsClient, refreshProjects, t, workspaceKind]
  );

  const onPickProjectLocation = useCallback(async () => {
    try {
      const selected = await open({
        directory: true,
        multiple: false,
        title: t("projects.pickFolderTitle")
      });
      if (!selected) {
        return;
      }

      const pickedLocation = Array.isArray(selected) ? selected[0] : selected;
      if (!pickedLocation || typeof pickedLocation !== "string") {
        return;
      }

      await onCreateProjectFromLocation(pickedLocation);
    } catch (error) {
      const normalized = normalizeUnknownError(error);
      addToast({
        title: t("projects.pickerFailed"),
        description: `${normalized.message} (${normalized.code})`,
        diagnostic: normalized.detail ?? normalized.message,
        variant: "error"
      });
    }
  }, [addToast, onCreateProjectFromLocation, t]);

  const onLogout = async () => {
    if (!currentProfile || currentProfile.kind !== "remote") {
      return;
    }

    try {
      await deleteToken(currentProfile.id);
      clearRemoteAuth(currentProfile.id);
      addToast({
        title: t("workspace.logoutSuccess"),
        variant: "success"
      });
    } catch (error) {
      addToast({
        title: t("workspace.logoutFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  const remoteUser = currentProfile?.kind === "remote" ? remoteUsersByProfileId[currentProfile.id] : undefined;
  const accountPrimary =
    currentProfile?.kind === "remote"
      ? remoteUser?.display_name ?? currentProfile.name
      : t("workspace.localAccount");
  const accountSecondary =
    currentProfile?.kind === "remote" ? remoteUser?.email ?? currentProfile.remote?.serverUrl ?? "" : t("workspace.localGroup");

  return (
    <>
      <aside
        className={cn(
          "flex h-full flex-col border-r border-border-subtle bg-muted/40 p-3 transition-[width] duration-150",
          collapsed ? "w-sidebar-collapsed" : "w-sidebar"
        )}
      >
        <div className="mb-3 flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <WorkspaceSwitcher collapsed={collapsed} />
          </div>
          <button
            type="button"
            className="inline-flex h-8 w-8 items-center justify-center rounded-control text-small text-muted-foreground hover:bg-muted"
            onClick={toggleSidebar}
            aria-label={collapsed ? t("app.sidebar.expand") : t("app.sidebar.collapse")}
          >
            {collapsed ? ">" : "<"}
          </button>
        </div>

        <div className="mb-3">
          <Button
            className={cn("w-full", collapsed ? "px-0" : "justify-start")}
            variant="outline"
            size="sm"
            onClick={() => {
              if (selectedProjectId) {
                void onNewThread(selectedProjectId);
              }
            }}
            disabled={!selectedProjectId}
          >
            <Sparkles className="h-4 w-4" />
            {!collapsed ? <span>{t("conversation.newThread")}</span> : null}
          </Button>
        </div>

        <div className="mb-2 flex items-center justify-between px-1 text-small text-muted-foreground">
          {!collapsed ? <span>{t("projects.title")}</span> : null}
          <button
            type="button"
            className="inline-flex h-6 w-6 items-center justify-center rounded-control hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="add-project"
            disabled={creatingProject}
            onClick={() => {
              void onPickProjectLocation();
            }}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-1 overflow-auto scrollbar-subtle">
          {loadingProjects && !collapsed ? <p className="px-2 text-small text-muted-foreground">{t("projects.loading")}</p> : null}

          {projects.map((project) => (
            <SidebarProjectSection
              key={project.project_id}
              collapsed={collapsed}
              project={project}
              sessions={sessionsByProjectId[project.project_id] ?? []}
              selectedProjectId={selectedProjectId}
              selectedSessionId={selectedSessionId}
              onSelectProject={setSelectedProject}
              onSelectSession={setSelectedSession}
              onNewThread={(projectId) => {
                void onNewThread(projectId);
              }}
            />
          ))}

          {!collapsed && extensionItems.length > 0 ? (
            <div className="mt-4 space-y-1 border-t border-border-subtle pt-3">
              <p className="px-2 text-small text-muted-foreground">{t("workspace.extensions")}</p>
              {extensionItems.map((item) => {
                const supported = item.route === "/settings";
                return (
                  <button
                    key={`${item.route}:${item.labelKey}`}
                    type="button"
                    className={cn(
                      "flex w-full items-center gap-2 rounded-control px-2 py-1.5 text-left text-small",
                      supported
                        ? "text-muted-foreground hover:bg-muted hover:text-foreground"
                        : "cursor-not-allowed text-muted-foreground/60"
                    )}
                    disabled={!supported}
                    onClick={() => navigate(item.route)}
                  >
                    <item.icon className="h-4 w-4" />
                    <span className="truncate">{t(item.labelKey)}</span>
                  </button>
                );
              })}
            </div>
          ) : null}
        </div>

        <div className="mt-3">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                size="sm"
                variant="ghost"
                className={cn("w-full", collapsed ? "px-0" : "justify-start")}
              >
                <Settings2 className="h-4 w-4" />
                {!collapsed ? <span>{t("nav.settings")}</span> : null}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              side="top"
              align={collapsed ? "center" : "start"}
              sideOffset={8}
              className="w-56"
            >
              <div className="rounded-control border border-border-subtle bg-muted/50 p-2">
                <div className="flex items-center gap-2">
                  <UserCircle2 className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="truncate text-small text-foreground">{accountPrimary}</p>
                    {accountSecondary ? (
                      <p className="truncate text-small text-muted-foreground">{accountSecondary}</p>
                    ) : null}
                  </div>
                </div>
              </div>

              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => navigate("/settings")}>
                <Settings2 className="mr-2 h-4 w-4" />
                <span>{t("nav.settings")}</span>
              </DropdownMenuItem>
              {currentProfile?.kind === "remote" ? (
                <DropdownMenuItem
                  onClick={() => {
                    void onLogout();
                  }}
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>{t("workspace.logout")}</span>
                </DropdownMenuItem>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </aside>
    </>
  );
}
