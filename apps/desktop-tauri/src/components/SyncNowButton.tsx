import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toast";

import { runSyncNow } from "../api/syncClient";

interface SyncNowButtonProps {
  compact?: boolean;
}

export function SyncNowButton({ compact = false }: SyncNowButtonProps) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<string>("");
  const { addToast } = useToast();

  const onClick = async () => {
    try {
      const result = await runSyncNow();
      const statusText = t("sync.status", { pushed: result.pushed, pulled: result.pulled });
      setStatus(statusText);
      addToast({
        title: t("sync.completedTitle"),
        description: statusText,
        variant: "success"
      });
    } catch (error) {
      setStatus((error as Error).message);
      addToast({
        title: t("sync.failedTitle"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  if (compact) {
    return (
      <Button size="sm" variant="outline" onClick={() => void onClick()}>
        {t("sync.syncNowCompact")}
      </Button>
    );
  }

  return (
    <div className="space-y-2">
      <Button variant="outline" onClick={() => void onClick()}>
        {t("sync.syncNow")}
      </Button>
      {status ? <p className="text-small text-muted-foreground">{status}</p> : null}
    </div>
  );
}
