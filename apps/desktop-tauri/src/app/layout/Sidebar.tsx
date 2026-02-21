import { open } from "@tauri-apps/plugin-dialog";
import type { LucideIcon } from "lucide-react";
import {
  AlertCircle,
  Circle,
  FolderKanban,
  FolderOpen,
  GitBranch,
  Loader2,
  LogOut,
  MessageSquare,
  Plus,
  RefreshCw,
  Settings2,
  Sparkles,
  Trash2,
  UserCircle2
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

import { type DataProject, getProjectsClient } from "@/api/dataSource";
import { deleteToken } from "@/api/secretStoreClient";
import { getSessionDataSource } from "@/api/sessionDataSource";
import { WorkspaceSwitcher } from "@/app/layout/WorkspaceSwitcher";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
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
  onDeleteSession?: (projectId: string, session: ConversationSessionSummary) => void;
  onRemoveProject?: (project: DataProject) => void;
  onSync?: (projectId: string) => void;
  isSyncing?: boolean;
}

function SyncStatusIcon({ status }: { status: DataProject["sync_status"] }) {
  if (status === "syncing") return <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />;
  if (status === "error") return <AlertCircle className="h-3 w-3 text-destructive" />;
  if (status === "pending") return <Loader2 className="h-3 w-3 text-muted-foreground/60" />;
  return null;
}

function SidebarProjectSection({
  collapsed,
  project,
  sessions,
  selectedProjectId,
  selectedSessionId,
  onSelectProject,
  onSelectSession,
  onNewThread,
  onDeleteSession,
  onRemoveProject,
  onSync,
  isSyncing
}: SidebarProjectSectionProps) {
  const { t } = useTranslation();
  const isActiveProject = selectedProjectId === project.project_id;
  const isGitProject = !!project.repo_url;

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
          {!collapsed ? (
            <>
              <span className="truncate">{project.name}</span>
              {isGitProject ? <SyncStatusIcon status={project.sync_status} /> : null}
            </>
          ) : null}
        </button>
        {!collapsed ? (
          <div className="flex items-center gap-0.5">
            {isGitProject && onSync ? (
              <button
                type="button"
                className="inline-flex h-6 w-6 items-center justify-center rounded-control text-muted-foreground hover:bg-background hover:text-foreground disabled:opacity-40"
                onClick={() => onSync(project.project_id)}
                disabled={isSyncing || project.sync_status === "syncing"}
                aria-label="sync-project"
              >
                <RefreshCw className="h-3 w-3" />
              </button>
            ) : null}
            <button
              type="button"
              className="inline-flex h-6 w-6 items-center justify-center rounded-control text-muted-foreground hover:bg-background hover:text-foreground"
              onClick={() => onNewThread(project.project_id)}
              aria-label="new-thread"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
            {onRemoveProject ? (
              <button
                type="button"
                className="inline-flex h-6 w-6 items-center justify-center rounded-control text-muted-foreground hover:bg-background hover:text-destructive"
                onClick={() => onRemoveProject(project)}
                aria-label="remove-project"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            ) : null}
          </div>
        ) : null}
      </div>

      {!collapsed && isActiveProject ? (
        <div className="ml-5 space-y-1 border-l border-border-subtle pl-2">
          {isGitProject && project.sync_error ? (
            <p className="px-2 text-xs text-destructive" title={project.sync_error}>
              {project.sync_error}
            </p>
          ) : null}
          {sessions.map((session) => (
            <div key={session.session_id} className="group flex items-center gap-1">
              <button
                type="button"
                className={cn(
                  "flex min-w-0 flex-1 items-center gap-2 rounded-control px-2 py-1 text-left text-small transition-colors",
                  selectedSessionId === session.session_id
                    ? "bg-accent/20 text-accent"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
                onClick={() => onSelectSession(project.project_id, session.session_id)}
              >
                <MessageSquare className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">{session.title}</span>
              </button>
              {onDeleteSession ? (
                <button
                  type="button"
                  className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-control text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-destructive group-hover:opacity-100"
                  aria-label={t("conversation.deleteAction")}
                  title={t("conversation.deleteAction")}
                  onClick={(event) => {
                    event.stopPropagation();
                    onDeleteSession(project.project_id, session);
                  }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              ) : null}
            </div>
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
  const removeSession = useConversationStore((state) => state.removeSession);
  const upsertSession = useConversationStore((state) => state.upsertSession);

  const projectsClient = useMemo(() => getProjectsClient(currentProfile), [currentProfile]);
  const sessionDataSource = useMemo(() => getSessionDataSource(currentProfile), [currentProfile]);

  const [projects, setProjects] = useState<DataProject[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [creatingProject, setCreatingProject] = useState(false);
  const [syncingProjectId, setSyncingProjectId] = useState<string | null>(null);

  // Git repo add dialog
  const [gitDialogOpen, setGitDialogOpen] = useState(false);
  const [gitRepoUrl, setGitRepoUrl] = useState("");
  const [gitBranch, setGitBranch] = useState("");
  const [gitName, setGitName] = useState("");

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

    sessionDataSource
      .listSessions(selectedProjectId)
      .then((payload) => setSessions(selectedProjectId, payload.sessions))
      .catch(() => {
        // Phase-1 fallback: keep local cache if session API is temporarily unavailable.
      });
  }, [selectedProjectId, sessionDataSource, setSessions]);

  const onNewThread = async (projectId: string) => {
    try {
      const payload = await sessionDataSource.createSession({
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

  const onDeleteSession = useCallback(
    async (projectId: string, session: ConversationSessionSummary) => {
      const confirmed = window.confirm(t("conversation.deleteConfirm", { title: session.title }));
      if (!confirmed) {
        return;
      }

      try {
        await sessionDataSource.archiveSession(session.session_id);
        removeSession(projectId, session.session_id);
        addToast({
          title: t("conversation.deleteSuccess"),
          variant: "success"
        });
      } catch (error) {
        addToast({
          title: t("conversation.deleteFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [addToast, removeSession, sessionDataSource, t]
  );

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

  const onCreateGitProject = useCallback(async () => {
    const repoUrl = gitRepoUrl.trim();
    if (!repoUrl) return;
    const name = gitName.trim() || deriveProjectName(repoUrl, t("projects.defaultName"));
    const branch = gitBranch.trim() || undefined;
    setCreatingProject(true);
    try {
      await projectsClient.createGit({ name, repo_url: repoUrl, branch });
      setGitDialogOpen(false);
      setGitRepoUrl("");
      setGitBranch("");
      setGitName("");
      await refreshProjects();
      addToast({ title: t("projects.createSuccess"), description: name, variant: "success" });
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
  }, [addToast, gitBranch, gitName, gitRepoUrl, projectsClient, refreshProjects, t]);

  const onSyncProject = useCallback(async (projectId: string) => {
    setSyncingProjectId(projectId);
    try {
      await projectsClient.sync(projectId);
      // Brief polling: after 1s refresh to show syncing state
      setTimeout(() => { void refreshProjects(); }, 1000);
    } catch (error) {
      addToast({
        title: t("projects.syncFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setSyncingProjectId(null);
    }
  }, [addToast, projectsClient, refreshProjects, t]);

  const onRemoveProject = useCallback(
    async (project: DataProject) => {
      if (!projectsClient.supportsDelete) {
        return;
      }

      const confirmed = window.confirm(t("projects.removeConfirm", { name: project.name }));
      if (!confirmed) {
        return;
      }

      try {
        await projectsClient.delete(project.project_id);
        await refreshProjects();
        addToast({
          title: t("projects.removeSuccess"),
          description: project.name,
          variant: "success"
        });
      } catch (error) {
        addToast({
          title: t("projects.removeFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [addToast, projectsClient, refreshProjects, t]
  );

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
        <div className="mb-3">
          <WorkspaceSwitcher collapsed={collapsed} />
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
          {projectsClient.supportsGit ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="inline-flex h-6 w-6 items-center justify-center rounded-control hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                  aria-label="add-project"
                  disabled={creatingProject}
                >
                  <Plus className="h-3.5 w-3.5" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent side="right" align="start" className="w-44">
                <DropdownMenuItem onClick={() => { void onPickProjectLocation(); }}>
                  <FolderOpen className="mr-2 h-4 w-4" />
                  <span>{t("projects.addFolder")}</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setGitDialogOpen(true)}>
                  <GitBranch className="mr-2 h-4 w-4" />
                  <span>{t("projects.addGitRepo")}</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <button
              type="button"
              className="inline-flex h-6 w-6 items-center justify-center rounded-control hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
              aria-label="add-project"
              disabled={creatingProject}
              onClick={() => { void onPickProjectLocation(); }}
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          )}
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
              onNewThread={(projectId) => { void onNewThread(projectId); }}
              onDeleteSession={(projectId, session) => { void onDeleteSession(projectId, session); }}
              onRemoveProject={projectsClient.supportsDelete ? (project) => { void onRemoveProject(project); } : undefined}
              onSync={projectsClient.supportsGit ? (projectId) => { void onSyncProject(projectId); } : undefined}
              isSyncing={syncingProjectId === project.project_id}
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

      {/* Git repo add dialog */}
      <Dialog open={gitDialogOpen} onOpenChange={setGitDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("projects.addGitRepo")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div>
              <label className="mb-1 block text-small text-muted-foreground">{t("projects.gitRepoUrl")}</label>
              <Input
                placeholder="https://github.com/org/repo.git"
                value={gitRepoUrl}
                onChange={(e) => setGitRepoUrl(e.target.value)}
                autoFocus
              />
            </div>
            <div>
              <label className="mb-1 block text-small text-muted-foreground">{t("projects.gitBranch")}</label>
              <Input
                placeholder="main"
                value={gitBranch}
                onChange={(e) => setGitBranch(e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1 block text-small text-muted-foreground">{t("projects.nameOptional")}</label>
              <Input
                placeholder={t("projects.nameDerivedFromUrl")}
                value={gitName}
                onChange={(e) => setGitName(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGitDialogOpen(false)}>{t("workspace.cancel")}</Button>
            <Button onClick={() => { void onCreateGitProject(); }} disabled={!gitRepoUrl.trim() || creatingProject}>
              {creatingProject ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {t("projects.addGitRepo")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
