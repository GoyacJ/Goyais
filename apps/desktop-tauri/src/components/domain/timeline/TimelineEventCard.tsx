import { Copy, LoaderCircle, MoreHorizontal, Pin, RefreshCcw, UnfoldVertical } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/cn";
import { formatTimeByLocale } from "@/lib/format";
import { useSettingsStore } from "@/stores/settingsStore";
import type { RunEventViewModel } from "@/types/ui";

interface TimelineEventCardProps {
  event: RunEventViewModel;
  selected: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onSelect: () => void;
  onRetry?: () => void;
}

const MAX_VISIBLE_LINES = 120;

function badgeVariantForType(type: RunEventViewModel["type"]) {
  switch (type) {
    case "error":
      return "destructive" as const;
    case "done":
      return "success" as const;
    case "patch":
      return "info" as const;
    case "tool_call":
      return "warning" as const;
    default:
      return "secondary" as const;
  }
}

function summaryLabel(event: RunEventViewModel, t: (key: string, options?: Record<string, unknown>) => string) {
  const payload = event.payload as Record<string, unknown>;

  if (event.type === "tool_call") {
    return t("timeline.summary.toolCall", { toolName: event.toolName ?? String(payload.tool_name ?? "unknown") });
  }

  if (event.type === "tool_result") {
    return t("timeline.summary.toolResult", {
      callId: event.callId ?? String(payload.call_id ?? "unknown"),
      status: payload.ok === true ? t("timeline.result.ok") : t("timeline.result.error")
    });
  }

  if (event.type === "patch") {
    return t("timeline.summary.patch");
  }

  if (event.type === "error") {
    return t("timeline.summary.error", { message: String(payload.message ?? "unknown") });
  }

  if (event.type === "done") {
    return t("timeline.summary.done", { status: String(payload.status ?? t("timeline.result.finished")) });
  }

  if (event.type === "plan") {
    return String(payload.summary ?? t("timeline.summary.planDefault"));
  }

  return event.summary;
}

export function TimelineEventCard({ event, selected, collapsed, onToggleCollapsed, onSelect, onRetry }: TimelineEventCardProps) {
  const { t } = useTranslation();
  const locale = useSettingsStore((state) => state.locale);

  const lines = event.payloadText.split("\n");
  const shouldCollapse = lines.length > MAX_VISIBLE_LINES;
  const visibleText = collapsed && shouldCollapse ? lines.slice(0, MAX_VISIBLE_LINES).join("\n") : event.payloadText;
  const timeText = formatTimeByLocale(event.ts, locale) || event.ts;

  return (
    <article
      className={cn(
        "rounded-panel border border-border bg-muted/20 p-3",
        selected && "border-accent/50 bg-accent/10"
      )}
      onClick={onSelect}
    >
      <header className="mb-2 flex items-center gap-2">
        <Badge variant={badgeVariantForType(event.type)}>{t(`timeline.types.${event.type}`, { defaultValue: event.type })}</Badge>
        <span className="text-small text-muted-foreground">#{event.seq}</span>
        <span className="text-small text-muted-foreground">{timeText}</span>
        <span className="truncate text-small text-foreground">{summaryLabel(event, t)}</span>
        <div className="ml-auto flex items-center gap-1">
          {event.streamState === "streaming" || event.streamState === "waiting_confirmation" ? (
            <LoaderCircle className="h-3.5 w-3.5 animate-spin text-info" />
          ) : null}
          <Button
            size="sm"
            variant="ghost"
            aria-label={t("timeline.actions.copyPayload")}
            onClick={(clickEvent) => {
              clickEvent.stopPropagation();
              void navigator.clipboard.writeText(event.payloadText);
            }}
          >
            <Copy className="h-3.5 w-3.5" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            aria-label={collapsed ? t("timeline.actions.showMore") : t("timeline.actions.showLess")}
            onClick={(clickEvent) => {
              clickEvent.stopPropagation();
              onToggleCollapsed();
            }}
          >
            <UnfoldVertical className="h-3.5 w-3.5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" onClick={(clickEvent) => clickEvent.stopPropagation()}>
                <MoreHorizontal className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>{t("timeline.actions.title")}</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={(clickEvent) => {
                  clickEvent.stopPropagation();
                  void navigator.clipboard.writeText(event.payloadText);
                }}
              >
                <Copy className="mr-2 h-3.5 w-3.5" />
                {t("timeline.actions.copyPayload")}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={(clickEvent) => clickEvent.stopPropagation()}>
                <Pin className="mr-2 h-3.5 w-3.5" />
                {t("timeline.actions.pinItem")}
              </DropdownMenuItem>
              {onRetry ? (
                <DropdownMenuItem
                  onClick={(clickEvent) => {
                    clickEvent.stopPropagation();
                    onRetry();
                  }}
                >
                  <RefreshCcw className="mr-2 h-3.5 w-3.5" />
                  {t("timeline.actions.retry")}
                </DropdownMenuItem>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>
      <pre className="max-h-80 overflow-auto rounded-control border border-border-subtle bg-background/70 p-2 text-code leading-relaxed scrollbar-subtle">
        {visibleText}
      </pre>
      {shouldCollapse ? (
        <Button
          size="sm"
          variant="ghost"
          className="mt-2"
          onClick={(clickEvent) => {
            clickEvent.stopPropagation();
            onToggleCollapsed();
          }}
        >
          {collapsed ? t("timeline.actions.showMore") : t("timeline.actions.showLess")}
        </Button>
      ) : null}
    </article>
  );
}
