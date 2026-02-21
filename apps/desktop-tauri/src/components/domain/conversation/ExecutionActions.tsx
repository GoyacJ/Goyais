import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

interface ExecutionActionsProps {
  executionId: string;
  commitSha?: string;
  busy?: boolean;
  onCommit: (message: string) => void | Promise<void>;
  onExportPatch: () => void | Promise<void>;
  onDiscard: () => void | Promise<void>;
}

export function ExecutionActions({
  executionId,
  commitSha,
  busy = false,
  onCommit,
  onExportPatch,
  onDiscard
}: ExecutionActionsProps) {
  const { t } = useTranslation();
  const [message, setMessage] = useState("");

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle>{t("conversation.executionActionsTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-xs text-muted-foreground">
          {t("conversation.executionIdLabel")}: <span className="font-mono text-foreground">{executionId}</span>
        </p>
        <div className="flex items-center gap-2">
          <Input
            value={message}
            disabled={busy}
            placeholder={t("conversation.commitMessagePlaceholder")}
            onChange={(event) => setMessage(event.target.value)}
          />
          <Button size="sm" disabled={busy} onClick={() => void onCommit(message)}>
            {t("conversation.commitAction")}
          </Button>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" disabled={busy} onClick={() => void onExportPatch()}>
            {t("conversation.exportPatchAction")}
          </Button>
          <Button variant="destructive" size="sm" disabled={busy} onClick={() => void onDiscard()}>
            {t("conversation.discardAction")}
          </Button>
        </div>
        {commitSha ? (
          <p className="text-xs text-muted-foreground">
            {t("conversation.commitShaLabel")}: <span className="font-mono text-foreground">{commitSha}</span>
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}
