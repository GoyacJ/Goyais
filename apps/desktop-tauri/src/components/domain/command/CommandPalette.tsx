import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { primaryShortcutLabel } from "@/lib/shortcuts";
import { useSettingsStore } from "@/stores/settingsStore";

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface PaletteCommand {
  id: string;
  label: string;
  group: "project" | "model" | "tool" | "settings";
  hint?: string;
  action: () => void;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const setTheme = useSettingsStore((state) => state.setTheme);

  const commands = useMemo<PaletteCommand[]>(
    () => [
      {
        id: "conversation",
        label: t("commandPalette.commands.conversationLabel"),
        group: "project",
        hint: t("commandPalette.commands.conversationHint"),
        action: () => navigate("/")
      },
      {
        id: "settings",
        label: t("commandPalette.commands.settingsLabel"),
        group: "settings",
        hint: t("commandPalette.commands.settingsHint"),
        action: () => navigate("/settings")
      },
      { id: "theme-dark", label: t("commandPalette.commands.themeDark"), group: "settings", action: () => setTheme("dark") },
      { id: "theme-light", label: t("commandPalette.commands.themeLight"), group: "settings", action: () => setTheme("light") }
    ],
    [navigate, setTheme, t]
  );

  const filtered = commands.filter((command) => {
    if (!query.trim()) {
      return true;
    }

    const normalized = `${command.label} ${t(`commandPalette.group.${command.group}`)} ${command.hint ?? ""}`.toLowerCase();
    return normalized.includes(query.toLowerCase());
  });

  const recent = filtered.slice(0, 3);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="glass-overlay max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <span>{t("commandPalette.title")}</span>
            <Badge variant="secondary">{primaryShortcutLabel("k")}</Badge>
          </DialogTitle>
        </DialogHeader>
        <Input autoFocus value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("commandPalette.searchPlaceholder")} />
        <div className="space-y-3">
          <section>
            <p className="mb-1 text-small text-muted-foreground">{t("commandPalette.recent")}</p>
            <div className="space-y-1">
              {recent.map((command) => (
                <button
                  key={`recent-${command.id}`}
                  className="flex w-full items-center justify-between rounded-control border border-border-subtle bg-background/60 px-2 py-1.5 text-left text-small hover:bg-muted"
                  onClick={() => {
                    command.action();
                    onOpenChange(false);
                  }}
                >
                  <span>{command.label}</span>
                  <Badge variant="secondary">{t(`commandPalette.group.${command.group}`)}</Badge>
                </button>
              ))}
            </div>
          </section>

          <section>
            <p className="mb-1 text-small text-muted-foreground">{t("commandPalette.allCommands")}</p>
            <div className="max-h-64 space-y-1 overflow-auto rounded-control border border-border-subtle bg-background/40 p-1 scrollbar-subtle">
              {filtered.map((command) => (
                <button
                  key={command.id}
                  className="flex w-full items-center justify-between rounded-control px-2 py-1.5 text-left text-small hover:bg-muted"
                  onClick={() => {
                    command.action();
                    onOpenChange(false);
                  }}
                >
                  <div>
                    <p className="text-foreground">{command.label}</p>
                    {command.hint ? <p className="text-small text-muted-foreground">{command.hint}</p> : null}
                  </div>
                  <Badge variant="secondary">{t(`commandPalette.group.${command.group}`)}</Badge>
                </button>
              ))}
            </div>
          </section>
        </div>
      </DialogContent>
    </Dialog>
  );
}
