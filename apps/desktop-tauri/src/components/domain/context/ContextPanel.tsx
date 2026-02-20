import { FileText, Search, StickyNote } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface ContextItem {
  id: string;
  source: "file" | "retrieval" | "pasted";
  label: string;
  bytes: number;
  tokenEstimate: number;
}

interface ContextPanelProps {
  workspacePath: string;
  taskInput: string;
  eventsCount: number;
}

function sourceIcon(source: ContextItem["source"]) {
  if (source === "file") return <FileText className="h-3.5 w-3.5" />;
  if (source === "retrieval") return <Search className="h-3.5 w-3.5" />;
  return <StickyNote className="h-3.5 w-3.5" />;
}

export function ContextPanel({ workspacePath, taskInput, eventsCount }: ContextPanelProps) {
  const { t } = useTranslation();
  const [removedIds, setRemovedIds] = useState<string[]>([]);

  const items = useMemo<ContextItem[]>(
    () => [
      {
        id: "workspace",
        source: "file",
        label: workspacePath,
        bytes: workspacePath.length,
        tokenEstimate: Math.ceil(workspacePath.length / 4)
      },
      {
        id: "task",
        source: "pasted",
        label: taskInput.slice(0, 80),
        bytes: taskInput.length,
        tokenEstimate: Math.ceil(taskInput.length / 4)
      },
      {
        id: "event-window",
        source: "retrieval",
        label: t("context.latestEvents", { count: eventsCount }),
        bytes: eventsCount * 64,
        tokenEstimate: eventsCount * 18
      }
    ],
    [workspacePath, taskInput, eventsCount, t]
  );

  const visibleItems = items.filter((item) => !removedIds.includes(item.id));
  const totalTokens = visibleItems.reduce((sum, item) => sum + item.tokenEstimate, 0);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle>{t("context.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {visibleItems.map((item) => (
          <div key={item.id} className="rounded-control border border-border-subtle bg-background/60 p-2">
            <div className="mb-1 flex items-center gap-2 text-small text-foreground">
              {sourceIcon(item.source)}
              <span className="truncate">{item.label}</span>
            </div>
            <div className="flex items-center gap-2 text-small text-muted-foreground">
              <Badge variant="secondary">{t(`context.source.${item.source}`)}</Badge>
              <span>{t("context.bytes", { count: item.bytes })}</span>
              <span>{t("context.tokens", { count: item.tokenEstimate })}</span>
              <Button size="sm" variant="ghost" className="ml-auto" onClick={() => setRemovedIds((state) => [...state, item.id])}>
                {t("context.remove")}
              </Button>
            </div>
          </div>
        ))}

        <div className="rounded-control border border-warning/40 bg-warning/10 p-2 text-small text-warning">
          {t("context.budget", { count: totalTokens })}
        </div>
      </CardContent>
    </Card>
  );
}
