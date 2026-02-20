import { CircleCheck, CircleX, Moon, RefreshCcw, Sun } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { getRunDataSource } from "@/api/runDataSource";
import { SyncNowButton } from "@/components/SyncNowButton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import type { SupportedLocale } from "@/i18n/types";
import { useRunStore } from "@/stores/runStore";
import { useSettingsStore } from "@/stores/settingsStore";
import { useUiStore } from "@/stores/uiStore";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";

export function Topbar() {
  const { t } = useTranslation();
  const theme = useUiStore((state) => state.theme);
  const setTheme = useUiStore((state) => state.setTheme);
  const locale = useSettingsStore((state) => state.locale);
  const setLocale = useSettingsStore((state) => state.setLocale);
  const context = useRunStore((state) => state.context);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const runDataSource = useMemo(() => getRunDataSource(currentProfile), [currentProfile]);
  const executionKind = currentProfile?.kind ?? "local";
  const executionTarget =
    executionKind === "remote" ? currentProfile?.remote?.serverUrl ?? "n/a" : "http://127.0.0.1:8040";
  const [runtimeOk, setRuntimeOk] = useState<boolean | null>(null);
  const [checking, setChecking] = useState(false);

  const checkRuntime = useCallback(async () => {
    setChecking(true);
    try {
      const payload = await runDataSource.runtimeHealth();
      setRuntimeOk(Boolean(payload.ok));
    } catch {
      setRuntimeOk(false);
    } finally {
      setChecking(false);
    }
  }, [runDataSource]);

  useEffect(() => {
    void checkRuntime();
  }, [checkRuntime]);

  return (
    <header className="flex h-toolbar items-center justify-between border-b border-border-subtle px-page">
      <div className="flex items-center gap-2 text-small">
        <Badge variant="secondary">
          {t("app.topbar.project")}: {context.projectId}
        </Badge>
        <Badge variant="secondary">
          {t("app.topbar.model")}: {context.modelConfigId}
        </Badge>
        <Badge variant="outline">
          {executionKind === "remote" ? t("app.topbar.remote") : t("app.topbar.local")}
        </Badge>
        <Badge variant="secondary">
          {t("app.topbar.endpoint")}: {executionTarget}
        </Badge>
        <Button size="sm" variant="ghost" onClick={() => void checkRuntime()}>
          <RefreshCcw className="h-3.5 w-3.5" />
          {checking ? t("app.topbar.checking") : t("app.topbar.runtime")}
        </Button>
        {runtimeOk === null ? null : runtimeOk ? (
          <Badge variant="success">
            <CircleCheck className="mr-1 h-3.5 w-3.5" />
            {t("app.topbar.online")}
          </Badge>
        ) : (
          <Badge variant="destructive">
            <CircleX className="mr-1 h-3.5 w-3.5" />
            {t("app.topbar.offline")}
          </Badge>
        )}
      </div>
      <div className="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="outline" aria-label={t("app.topbar.language")}>
              {t("app.locale." + locale)}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {(["zh-CN", "en-US"] satisfies SupportedLocale[]).map((item) => (
              <DropdownMenuItem key={item} onClick={() => void setLocale(item)}>
                {t("app.locale." + item)}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
        <SyncNowButton compact />
        <Button
          size="icon"
          variant="ghost"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          aria-label={t("app.topbar.toggleTheme")}
        >
          {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
        </Button>
      </div>
    </header>
  );
}
