import { type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

interface RunComposerValues {
  projectId: string;
  sessionId: string;
  modelConfigId: string;
  workspacePath: string;
  taskInput: string;
}

interface RunComposerPanelProps {
  values: RunComposerValues;
  running: boolean;
  onChange: (next: Partial<RunComposerValues>) => void;
  onSubmit: (event: FormEvent) => void;
}

export function RunComposerPanel({ values, running, onChange, onSubmit }: RunComposerPanelProps) {
  const { t } = useTranslation();

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>{t("run.composer.title")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form className="space-y-form" onSubmit={onSubmit}>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("run.composer.projectId")}
            <Input value={values.projectId} onChange={(event) => onChange({ projectId: event.target.value })} />
          </label>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("run.composer.sessionId")}
            <Input value={values.sessionId} onChange={(event) => onChange({ sessionId: event.target.value })} />
          </label>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("run.composer.modelConfigId")}
            <Input value={values.modelConfigId} onChange={(event) => onChange({ modelConfigId: event.target.value })} />
          </label>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("run.composer.workspacePath")}
            <Input value={values.workspacePath} onChange={(event) => onChange({ workspacePath: event.target.value })} />
          </label>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("run.composer.task")}
            <Textarea rows={6} value={values.taskInput} onChange={(event) => onChange({ taskInput: event.target.value })} />
          </label>
          <Button className="w-full" type="submit" disabled={running}>
            {running ? t("run.composer.running") : t("run.composer.startRun")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
