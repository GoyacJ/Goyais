import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatTimeByLocale } from "@/lib/format";
import { classifyToolRisk } from "@/lib/risk";
import type { PendingConfirmation } from "@/stores/runStore";
import { useSettingsStore } from "@/stores/settingsStore";

interface PermissionQueueCenterProps {
  queue: PendingConfirmation[];
  onOpen: (callId: string) => void;
}

export function PermissionQueueCenter({ queue, onOpen }: PermissionQueueCenterProps) {
  const { t } = useTranslation();
  const locale = useSettingsStore((state) => state.locale);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle>{t("permission.queueTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {queue.length === 0 ? (
          <p className="text-small text-muted-foreground">{t("permission.noPending")}</p>
        ) : (
          queue.map((item) => {
            const risk = classifyToolRisk(item.toolName, item.args);
            return (
              <div key={item.callId} className="rounded-control border border-border-subtle bg-background/60 p-2">
                <div className="flex items-center gap-2 text-small">
                  <span className="font-medium text-foreground">{item.toolName}</span>
                  <Badge variant="warning">{t("permission.riskLabel", { risk: risk.primary })}</Badge>
                  <span className="ml-auto text-muted-foreground">{formatTimeByLocale(item.createdAt, locale) || item.createdAt}</span>
                </div>
                <div className="mt-2 flex items-center gap-2">
                  <Button size="sm" onClick={() => onOpen(item.callId)}>
                    {t("permission.review")}
                  </Button>
                </div>
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}
