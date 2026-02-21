import {
  ChevronDown,
  Copy,
  GitCommitHorizontal,
  PanelRight,
  ShieldCheck,
  Square
} from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";

import { cn } from "@/lib/cn";
import { useConversationStore } from "@/stores/conversationStore";
import { useUiStore } from "@/stores/uiStore";

export function AppTopBar() {
  const { t } = useTranslation();
  const location = useLocation();
  const sidebarCollapsed = useUiStore((state) => state.sidebarCollapsed);

  const selectedProjectId = useConversationStore((state) => state.selectedProjectId);
  const selectedSessionId = useConversationStore((state) => state.selectedSessionId);
  const sessionsByProjectId = useConversationStore((state) => state.sessionsByProjectId);

  const isSettingsRoute = location.pathname === "/settings" || location.pathname.startsWith("/settings/");
  const selectedSession = useMemo(
    () => sessionsByProjectId[selectedProjectId ?? ""]?.find((item) => item.session_id === selectedSessionId),
    [selectedProjectId, selectedSessionId, sessionsByProjectId]
  );
  const title = isSettingsRoute ? t("nav.settings") : selectedSession?.title ?? t("conversation.newThread");

  return (
    <header className="flex h-11 shrink-0 items-center border-b border-border-subtle bg-background/95">
      <div
        className={cn(
          "flex h-full items-center gap-3 border-r border-border-subtle px-4",
          isSettingsRoute ? "w-sidebar" : sidebarCollapsed ? "w-sidebar-collapsed" : "w-sidebar"
        )}
      >
        <div className="flex items-center gap-2" aria-hidden>
          <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
          <span className="h-3 w-3 rounded-full bg-[#febc2e]" />
          <span className="h-3 w-3 rounded-full bg-[#28c840]" />
        </div>
        <button
          type="button"
          className={cn(
            "inline-flex h-5 w-5 items-center justify-center rounded-md border border-border-subtle text-muted-foreground",
            isSettingsRoute ? "" : sidebarCollapsed ? "opacity-0" : "opacity-100"
          )}
          aria-label="window-layout"
        >
          <Square className="h-3 w-3" />
        </button>
      </div>

      <div className="flex min-w-0 flex-1 items-center justify-between gap-3 px-4">
        <p className="truncate text-small font-semibold text-foreground">{title}</p>

        <div className="flex items-center gap-1">
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1 rounded-control border border-border-subtle bg-muted/40 px-3 text-small text-foreground hover:bg-muted/70"
          >
            <ShieldCheck className="h-3.5 w-3.5 text-muted-foreground" />
            <span>Open</span>
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          </button>
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1 rounded-control border border-border-subtle bg-muted/40 px-3 text-small text-foreground hover:bg-muted/70"
          >
            <GitCommitHorizontal className="h-3.5 w-3.5 text-muted-foreground" />
            <span>Commit</span>
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          </button>
          <div className="mx-1 h-5 w-px bg-border-subtle" />
          <button
            type="button"
            className="inline-flex h-8 w-8 items-center justify-center rounded-control text-muted-foreground hover:bg-muted/70 hover:text-foreground"
            aria-label="toggle-right-panel"
          >
            <PanelRight className="h-4 w-4" />
          </button>
          <button
            type="button"
            className="inline-flex h-8 w-8 items-center justify-center rounded-control text-muted-foreground hover:bg-muted/70 hover:text-foreground"
            aria-label="duplicate-conversation"
          >
            <Copy className="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>
  );
}
