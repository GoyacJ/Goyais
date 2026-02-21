import { ArrowUp, Plus } from "lucide-react";
import { type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";

interface ExecutionComposerPanelProps {
  input: string;
  modelConfigId?: string;
  modelOptions: Array<{ model_config_id: string; provider: string; model: string }>;
  running: boolean;
  sessionMode: "plan" | "agent";
  onInputChange: (value: string) => void;
  onModelChange: (value: string | undefined) => void;
  onSessionModeChange: (mode: "plan" | "agent") => void;
  onSubmit: (event: FormEvent) => void;
}

export function ExecutionComposerPanel({
  input,
  modelConfigId,
  modelOptions,
  running,
  sessionMode,
  onInputChange,
  onModelChange,
  onSessionModeChange,
  onSubmit
}: ExecutionComposerPanelProps) {
  const { t } = useTranslation();

  return (
    <div className="bg-muted/10 p-1">
      <form onSubmit={onSubmit}>
        <div className="relative overflow-hidden rounded-panel border border-border-subtle bg-background/70">
          <Textarea
            rows={3}
            className="min-h-[96px] max-h-44 resize-y border-0 bg-transparent pb-12 pr-14 focus-visible:ring-0 focus-visible:ring-offset-0"
            placeholder={t("conversation.inputPlaceholder")}
            value={input}
            onChange={(event) => onInputChange(event.target.value)}
          />

          <div className="absolute inset-x-0 bottom-0 flex items-center justify-between gap-2 px-2 pb-2">
            <div className="flex min-w-0 items-center gap-2">
              <span className="text-xs text-muted-foreground">{t("conversation.sessionMode")}:</span>
              <select
                className="h-7 min-w-[5.5rem] rounded-control border-0 bg-muted/70 px-2 text-small text-foreground focus-visible:outline-none"
                value={sessionMode}
                onChange={(event) => onSessionModeChange(event.target.value as "plan" | "agent")}
              >
                <option value="agent">{t("conversation.sessionModeAgent")}</option>
                <option value="plan">{t("conversation.sessionModePlan")}</option>
              </select>

              <div className="min-w-0">
                <select
                  className="h-7 max-w-[15rem] truncate rounded-control border-0 bg-muted/70 px-2 text-small text-foreground focus-visible:outline-none"
                  value={modelConfigId ?? ""}
                  onChange={(event) => onModelChange(event.target.value || undefined)}
                >
                  <option value="">{t("settings.noDefaultModel")}</option>
                  {modelOptions.map((item) => (
                    <option key={item.model_config_id} value={item.model_config_id}>
                      {item.provider}:{item.model}
                    </option>
                  ))}
                </select>
              </div>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className="inline-flex h-7 w-7 items-center justify-center rounded-control border border-border-subtle bg-muted/70 text-muted-foreground hover:bg-muted hover:text-foreground"
                    aria-label={t("conversation.quickActions")}
                    title={t("conversation.quickActions")}
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" side="top">
                  <DropdownMenuItem disabled>{t("conversation.quickActionComingSoon")}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <Button
              type="submit"
              size="icon"
              className="h-8 w-8 rounded-full"
              disabled={running || !input.trim()}
              aria-label={running ? t("conversation.sending") : t("conversation.send")}
              title={running ? t("conversation.sending") : t("conversation.send")}
            >
              <ArrowUp className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
