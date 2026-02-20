import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { classifyToolRisk } from "@/lib/risk";
import type { ToolCallView } from "@/types/ui";

interface ToolDetailsDrawerProps {
  toolCalls: Record<string, ToolCallView>;
  selectedCallId?: string;
  onSelectCallId: (callId: string) => void;
}

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2);
}

function statusBadgeVariant(status: ToolCallView["status"]) {
  if (status === "failed" || status === "denied") return "destructive" as const;
  if (status === "waiting" || status === "approved") return "warning" as const;
  if (status === "completed") return "success" as const;
  return "secondary" as const;
}

export function ToolDetailsDrawer({ toolCalls, selectedCallId, onSelectCallId }: ToolDetailsDrawerProps) {
  const { t } = useTranslation();
  const orderedCalls = useMemo(
    () => Object.values(toolCalls).sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()),
    [toolCalls]
  );
  const selected = selectedCallId ? toolCalls[selectedCallId] : orderedCalls[0];
  const risk = selected ? classifyToolRisk(selected.toolName, selected.args) : null;

  return (
    <Card className="h-full min-h-[18rem]">
      <CardHeader className="pb-2">
        <CardTitle>{t("toolDetails.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {orderedCalls.length === 0 ? (
          <p className="text-small text-muted-foreground">{t("toolDetails.noCalls")}</p>
        ) : (
          <>
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("toolDetails.toolCall")}
              <select
                className="h-10 rounded-control border border-border bg-background px-2 text-body"
                value={selected?.callId}
                onChange={(event) => onSelectCallId(event.target.value)}
              >
                {orderedCalls.map((call) => (
                  <option key={call.callId} value={call.callId}>
                    {t("toolDetails.toolCallOption", { toolName: call.toolName, callId: call.callId })}
                  </option>
                ))}
              </select>
            </label>

            {selected ? (
              <>
                <div className="flex flex-wrap items-center gap-2 text-small">
                  <Badge variant="secondary">{selected.toolName}</Badge>
                  <Badge variant={statusBadgeVariant(selected.status)}>{selected.status}</Badge>
                  {risk?.hasRisk ? <Badge variant="warning">{t("permission.riskLabel", { risk: risk.primary })}</Badge> : null}
                  <Button size="sm" variant="ghost" onClick={() => void navigator.clipboard.writeText(formatJson(selected))}>
                    {t("toolDetails.copyDiagnostics")}
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => undefined}>
                    {t("toolDetails.exportDiagnostics")}
                  </Button>
                </div>

                <Tabs defaultValue="input" className="w-full">
                  <TabsList>
                    <TabsTrigger value="input">{t("toolDetails.tabs.input")}</TabsTrigger>
                    <TabsTrigger value="output">{t("toolDetails.tabs.output")}</TabsTrigger>
                    <TabsTrigger value="logs">{t("toolDetails.tabs.logs")}</TabsTrigger>
                    <TabsTrigger value="timing">{t("toolDetails.tabs.timing")}</TabsTrigger>
                  </TabsList>
                  <TabsContent value="input">
                    <ScrollArea className="h-48 rounded-control border border-border-subtle bg-background/70 p-2">
                      <pre className="text-code">{formatJson(selected.args)}</pre>
                    </ScrollArea>
                  </TabsContent>
                  <TabsContent value="output">
                    <ScrollArea className="h-48 rounded-control border border-border-subtle bg-background/70 p-2">
                      <pre className="text-code">{formatJson(selected.output ?? { message: t("toolDetails.noOutputYet") })}</pre>
                    </ScrollArea>
                  </TabsContent>
                  <TabsContent value="logs">
                    <ScrollArea className="h-48 rounded-control border border-border-subtle bg-background/70 p-2">
                      <pre className="text-code">{formatJson({ status: selected.status, risk: risk?.details })}</pre>
                    </ScrollArea>
                  </TabsContent>
                  <TabsContent value="timing">
                    <ScrollArea className="h-48 rounded-control border border-border-subtle bg-background/70 p-2">
                      <pre className="text-code">
                        {formatJson({
                          createdAt: selected.createdAt,
                          finishedAt: selected.finishedAt ?? t("toolDetails.pending")
                        })}
                      </pre>
                    </ScrollArea>
                  </TabsContent>
                </Tabs>
              </>
            ) : null}
          </>
        )}
      </CardContent>
    </Card>
  );
}
