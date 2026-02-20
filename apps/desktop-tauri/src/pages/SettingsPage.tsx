import { useTranslation } from "react-i18next";

import { SyncNowButton } from "@/components/SyncNowButton";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { SupportedLocale } from "@/i18n/types";
import { useSettingsStore } from "@/stores/settingsStore";

export function SettingsPage() {
  const { t } = useTranslation();
  const locale = useSettingsStore((state) => state.locale);
  const setLocale = useSettingsStore((state) => state.setLocale);

  return (
    <div className="grid max-w-2xl gap-panel">
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.title")}</CardTitle>
          <CardDescription>{t("settings.description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-control border border-border-subtle bg-background/60 p-3">
            <p className="mb-2 text-small text-muted-foreground">{t("settings.syncSection")}</p>
            <SyncNowButton />
          </div>
          <div className="rounded-control border border-border-subtle bg-background/60 p-3">
            <label className="grid gap-1 text-small text-muted-foreground">
              {t("settings.languageLabel")}
              <select
                className="h-10 rounded-control border border-border bg-background px-2 text-body text-foreground"
                value={locale}
                onChange={(event) => void setLocale(event.target.value as SupportedLocale)}
              >
                {(["zh-CN", "en-US"] satisfies SupportedLocale[]).map((item) => (
                  <option key={item} value={item}>
                    {t("app.locale." + item)}
                  </option>
                ))}
              </select>
              <span className="text-small text-muted-foreground">{t("settings.languageHelp")}</span>
            </label>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
