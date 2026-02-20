import { FormEvent, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { createProject, listProjects } from "@/api/runtimeClient";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";

export function ProjectsPage() {
  const { t } = useTranslation();
  const [name, setName] = useState("Demo Project");
  const [workspacePath, setWorkspacePath] = useState("/Users/goya/Repo/Git/Goyais");
  const [projects, setProjects] = useState<Array<Record<string, string>>>([]);
  const [loading, setLoading] = useState(false);
  const { addToast } = useToast();

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await listProjects();
      setProjects(payload.projects);
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
  }, [addToast, t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      await createProject({ name, workspace_path: workspacePath });
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
              {t("projects.workspacePath")}
              <Input value={workspacePath} onChange={(event) => setWorkspacePath(event.target.value)} />
            </label>
            <Button className="w-full" type="submit">
              {t("projects.create")}
            </Button>
          </form>
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
                <p className="text-small text-muted-foreground">{project.workspace_path}</p>
                <p className="text-small text-muted-foreground">{t("projects.projectId", { id: project.project_id })}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
