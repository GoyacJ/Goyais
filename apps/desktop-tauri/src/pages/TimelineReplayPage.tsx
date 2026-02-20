import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { listRuns } from "@/api/runtimeClient";
import { LoadingState } from "@/components/domain/feedback/LoadingState";
import { TimelinePanel } from "@/components/domain/timeline/TimelinePanel";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { useReplay } from "@/hooks/useReplay";

export function ReplayPage() {
  const { t } = useTranslation();
  const [sessionId, setSessionId] = useState("session-demo");
  const [runs, setRuns] = useState<Array<Record<string, string>>>([]);
  const [runId, setRunId] = useState<string>();
  const [selectedEventId, setSelectedEventId] = useState<string>();
  const { addToast } = useToast();

  const { events, loading } = useReplay(runId);

  const onLoadRuns = async (event: FormEvent) => {
    event.preventDefault();
    try {
      const payload = await listRuns(sessionId);
      setRuns(payload.runs);
    } catch (error) {
      addToast({
        title: t("replay.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  return (
    <div className="grid gap-panel lg:grid-cols-[20rem_minmax(0,1fr)]">
      <Card>
        <CardHeader>
          <CardTitle>{t("replay.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form onSubmit={onLoadRuns} className="space-y-form">
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("replay.sessionId")}
              <Input value={sessionId} onChange={(event) => setSessionId(event.target.value)} />
            </label>
            <Button className="w-full" type="submit">
              {t("replay.loadRuns")}
            </Button>
          </form>

          <div className="space-y-2">
            {runs.map((run) => (
              <button
                key={run.run_id}
                className="w-full rounded-control border border-border-subtle bg-background/60 p-2 text-left text-small hover:bg-muted"
                onClick={() => setRunId(run.run_id)}
              >
                <p className="font-medium text-foreground">{run.run_id}</p>
                <p className="text-muted-foreground">{run.status}</p>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card className="h-[calc(100vh-12rem)] min-h-[30rem]">
        <CardHeader>
          <CardTitle>{runId ? t("replay.timelineRun", { runId }) : t("replay.timeline")}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-4.5rem)] min-h-0">
          {loading ? (
            <LoadingState label={t("replay.loading")} />
          ) : (
            <TimelinePanel events={events} selectedEventId={selectedEventId} onSelectEvent={setSelectedEventId} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
