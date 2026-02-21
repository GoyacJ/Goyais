import { CircleCheck, CircleX } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { getSessionDataSource, localHubBaseUrl } from "@/api/sessionDataSource";
import { Badge } from "@/components/ui/badge";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";

export function StatusBar() {
  const { t } = useTranslation();
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const sessionDataSource = useMemo(() => getSessionDataSource(currentProfile), [currentProfile]);
  const [runtimeOk, setRuntimeOk] = useState<boolean | null>(null);

  const checkRuntime = useCallback(async () => {
    try {
      const payload = await sessionDataSource.runtimeHealth();
      setRuntimeOk(Boolean(payload.ok));
    } catch {
      setRuntimeOk(false);
    }
  }, [sessionDataSource]);

  useEffect(() => {
    void checkRuntime();
    const timer = window.setInterval(() => {
      void checkRuntime();
    }, 30_000);

    return () => window.clearInterval(timer);
  }, [checkRuntime]);

  const executionKind = currentProfile?.kind ?? "local";
  const executionTarget =
    executionKind === "remote" ? currentProfile?.remote?.serverUrl ?? "n/a" : localHubBaseUrl();

  return (
    <footer className="flex h-8 items-center justify-between border-t border-border-subtle px-page text-small text-muted-foreground">
      <div className="flex items-center gap-2">
        <Badge variant="outline">{executionKind === "remote" ? t("app.topbar.remote") : t("app.topbar.local")}</Badge>
        <span className="truncate">{executionTarget}</span>
      </div>
      <div className="flex items-center gap-1">
        {runtimeOk === null ? null : runtimeOk ? (
          <>
            <CircleCheck className="h-3.5 w-3.5 text-success" />
            <span>{t("app.topbar.online")}</span>
          </>
        ) : (
          <>
            <CircleX className="h-3.5 w-3.5 text-destructive" />
            <span>{t("app.topbar.offline")}</span>
          </>
        )}
      </div>
    </footer>
  );
}
