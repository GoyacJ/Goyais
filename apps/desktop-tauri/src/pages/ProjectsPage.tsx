import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { DataProject, getProjectsClient } from "@/api/dataSource";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import {
  selectCurrentPermissions,
  selectCurrentProfile,
  selectCurrentWorkspaceKind,
  useWorkspaceStore
} from "@/stores/workspaceStore";

export function canWriteProjects(workspaceKind: "local" | "remote", permissions: string[]): boolean {
  if (workspaceKind === "local") {
    return true;
  }
  return permissions.includes("project:write");
}

export function ProjectsPage() {
  const { t } = useTranslation();
  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const projectsClient = useMemo(() => getProjectsClient(currentProfile), [currentProfile]);
  const writable = canWriteProjects(workspaceKind, permissions);

  const [name, setName] = useState("Demo Project");
  const [location, setLocation] = useState("/Users/goya/Repo/Git/Goyais");
  const [projects, setProjects] = useState<DataProject[]>([]);
  const [loading, setLoading] = useState(false);
  const { addToast } = useToast();

  useEffect(() => {
    setLocation(workspaceKind === "remote" ? "repo://demo/main" : "/Users/goya/Repo/Git/Goyais");
  }, [workspaceKind]);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      setProjects(await projectsClient.list());
    } catch (error) {
      addToast({
        title: t("projects.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setLoading(false);
    }
  }, [addToast, projectsClient, t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!writable) {
      return;
    }

    try {
      await projectsClient.create({ name, location });
      addToast({
        title: t("projects.createSuccess"),
        description: name,
        variant: "success"
      });
      await refresh();
    } catch (error) {
      addToast({
        title: t("projects.createFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  const onDelete = async (projectId: string) => {
    if (!projectsClient.supportsDelete || !writable) {
      return;
    }

    try {
      await projectsClient.delete(projectId);
      addToast({
        title: t("projects.deleteSuccess"),
        variant: "success"
      });
      await refresh();
    } catch (error) {
      addToast({
        title: t("projects.deleteFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  return (
    <div className="grid gap-panel lg:grid-cols-[22rem_minmax(0,1fr)]">
      <Card>
        <CardHeader>
          <CardTitle>{t("projects.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-form">
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("projects.name")}
              <Input value={name} onChange={(event) => setName(event.target.value)} />
            </label>
            <label className="grid gap-1 text-small text-muted-foreground">
              {workspaceKind === "remote" ? t("projects.rootUri") : t("projects.workspacePath")}
              <Input value={location} onChange={(event) => setLocation(event.target.value)} />
            </label>
            <Button className="w-full" type="submit" disabled={!writable}>
              {t("projects.create")}
            </Button>
          </form>
          {workspaceKind === "remote" ? <p className="mt-3 text-small text-muted-foreground">{t("projects.remoteHint")}</p> : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("projects.listTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? <p className="text-small text-muted-foreground">{t("projects.loading")}</p> : null}
          <div className="space-y-2">
            {projects.map((project) => (
              <div key={project.project_id} className="rounded-control border border-border-subtle bg-background/60 p-2">
                <p className="text-body font-medium text-foreground">{project.name}</p>
                <p className="text-small text-muted-foreground">{project.root_uri ?? project.workspace_path}</p>
                <p className="text-small text-muted-foreground">{t("projects.projectId", { id: project.project_id })}</p>
                {projectsClient.supportsDelete ? (
                  <div className="mt-2">
                    <Button
                      size="sm"
                      variant="destructive"
                      disabled={!writable}
                      onClick={() => void onDelete(project.project_id)}
                    >
                      {t("projects.delete")}
                    </Button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
