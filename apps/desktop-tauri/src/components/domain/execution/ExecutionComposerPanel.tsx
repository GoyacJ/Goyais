import { ArrowUp } from "lucide-react";
import { type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/cn";

interface ExecutionComposerPanelProps {
  input: string;
  modelConfigId?: string;
  modelOptions: Array<{ model_config_id: string; provider: string; model: string }>;
  running: boolean;
  sessionMode: "plan" | "agent";
  sessionUseWorktree: boolean;
  onInputChange: (value: string) => void;
  onModelChange: (value: string | undefined) => void;
  onSessionModeChange: (mode: "plan" | "agent") => void;
  onSessionUseWorktreeChange: (value: boolean) => void;
  onSubmit: (event: FormEvent) => void;
}

export function ExecutionComposerPanel({
  input,
  modelConfigId,
  modelOptions,
  running,
  sessionMode,
  sessionUseWorktree,
  onInputChange,
  onModelChange,
  onSessionModeChange,
  onSessionUseWorktreeChange,
  onSubmit
}: ExecutionComposerPanelProps) {
  const { t } = useTranslation();

  return (
    <div className="bg-muted/10 p-1.5">
      <form onSubmit={onSubmit}>
        <div className="relative overflow-hidden rounded-panel border border-border-subtle bg-background/70">
          <Textarea
            rows={3}
            className="min-h-[108px] max-h-44 resize-y border-0 bg-transparent pb-[4.5rem] pr-14 focus-visible:ring-0 focus-visible:ring-offset-0"
            placeholder={t("conversation.inputPlaceholder")}
            value={input}
            onChange={(event) => onInputChange(event.target.value)}
          />

          <div className="absolute inset-x-0 bottom-0 flex flex-col gap-1 px-2 pb-2">
            {/* Session settings row */}
            <div className="flex items-center gap-2">
              {/* Mode toggle */}
              <span className="text-xs text-muted-foreground">{t("conversation.sessionMode")}:</span>
              <div className="flex overflow-hidden rounded-control border border-border-subtle">
                <button
                  type="button"
                  title={t("conversation.sessionModeAgentDescription")}
                  className={cn(
                    "px-2 py-0.5 text-xs transition-colors",
                    sessionMode === "agent"
                      ? "bg-primary text-primary-foreground"
                      : "bg-transparent text-muted-foreground hover:bg-muted/70"
                  )}
                  onClick={() => onSessionModeChange("agent")}
                >
                  {t("conversation.sessionModeAgent")}
                </button>
                <button
                  type="button"
                  title={t("conversation.sessionModePlanDescription")}
                  className={cn(
                    "px-2 py-0.5 text-xs transition-colors",
                    sessionMode === "plan"
                      ? "bg-primary text-primary-foreground"
                      : "bg-transparent text-muted-foreground hover:bg-muted/70"
                  )}
                  onClick={() => onSessionModeChange("plan")}
                >
                  {t("conversation.sessionModePlan")}
                </button>
              </div>

              {/* Worktree toggle */}
              <span className="text-xs text-muted-foreground">{t("conversation.sessionWorktree")}:</span>
              <button
                type="button"
                title={sessionUseWorktree ? t("conversation.sessionWorktreeOn") : t("conversation.sessionWorktreeOff")}
                className={cn(
                  "rounded-control border px-2 py-0.5 text-xs transition-colors",
                  sessionUseWorktree
                    ? "border-primary/40 bg-primary/10 text-primary"
                    : "border-border-subtle bg-transparent text-muted-foreground hover:bg-muted/70"
                )}
                onClick={() => onSessionUseWorktreeChange(!sessionUseWorktree)}
              >
                {sessionUseWorktree ? t("conversation.sessionWorktreeOn") : t("conversation.sessionWorktreeOff")}
              </button>
            </div>

            {/* Model + Submit row */}
            <div className="flex items-center justify-between gap-2">
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
        </div>
      </form>
    </div>
  );
}
