import { useMemo, useReducer } from "react";
import DiffViewer from "react-diff-viewer-continued";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { parseUnifiedDiff, reduceDiffSelection } from "@/lib/diff";

interface DiffPanelProps {
  unifiedDiff?: string;
}

export function DiffPanel({ unifiedDiff }: DiffPanelProps) {
  const { t } = useTranslation();
  const parsed = useMemo(() => parseUnifiedDiff(unifiedDiff ?? ""), [unifiedDiff]);
  const [selection, dispatch] = useReducer(reduceDiffSelection, {});

  if (!unifiedDiff) {
    return (
      <Card className="h-full">
        <CardHeader>
          <CardTitle>{t("diff.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-small text-muted-foreground">{t("diff.noPatch")}</p>
        </CardContent>
      </Card>
    );
  }

  const selectedCount = Object.values(selection).filter(Boolean).length;
  const hasSelection = selectedCount > 0;

  return (
    <Card className="h-full min-h-0">
      <CardHeader className="pb-2">
        <CardTitle>{t("diff.titleStrategy")}</CardTitle>
      </CardHeader>
      <CardContent className="flex h-[calc(100%-4.5rem)] min-h-0 flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            onClick={() => {
              void navigator.clipboard.writeText(unifiedDiff);
            }}
            variant="outline"
          >
            {t("diff.copyRaw")}
          </Button>
          <Button size="sm" onClick={() => undefined}>
            {hasSelection ? t("diff.applySelected", { count: selectedCount }) : t("diff.apply")}
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => {
              const confirmed = window.confirm(t("diff.acceptConfirm"));
              if (confirmed) {
                dispatch({ type: "selectAll", hunkIds: parsed.hunks.map((hunk) => hunk.id) });
              }
            }}
          >
            {t("diff.acceptAll")}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => dispatch({ type: "clear" })}>
            {t("diff.clear")}
          </Button>
        </div>

        <ScrollArea className="min-h-0 flex-1 rounded-control border border-border-subtle bg-background/40">
          <div className="space-y-3 p-2">
            {parsed.hunks.map((hunk) => (
              <details key={hunk.id} open className="rounded-control border border-border-subtle bg-background/70 p-2">
                <summary className="flex cursor-pointer items-center gap-2 text-small text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={Boolean(selection[hunk.id])}
                    onChange={() => dispatch({ type: "toggle", hunkId: hunk.id })}
                  />
                  <span>{hunk.header}</span>
                </summary>
                <pre className="mt-2 overflow-x-auto rounded-control border border-border-subtle bg-background p-2 text-code leading-relaxed">
                  {hunk.lines.join("\n")}
                </pre>
              </details>
            ))}

            <div className="rounded-control border border-border-subtle bg-background p-2">
              <DiffViewer oldValue={parsed.oldText} newValue={parsed.newText} splitView showDiffOnly={false} />
            </div>
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
