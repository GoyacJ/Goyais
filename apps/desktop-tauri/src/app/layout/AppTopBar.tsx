import { getCurrentWindow } from "@tauri-apps/api/window";
import { type MouseEvent, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";

import { cn } from "@/lib/cn";
import { useConversationStore } from "@/stores/conversationStore";
import { useUiStore } from "@/stores/uiStore";

export function AppTopBar() {
  const { t } = useTranslation();
  const location = useLocation();
  const sidebarCollapsed = useUiStore((state) => state.sidebarCollapsed);
  const isMac = typeof navigator !== "undefined" && /Mac/i.test(navigator.platform);
  const appWindow = getCurrentWindow();

  const selectedProjectId = useConversationStore((state) => state.selectedProjectId);
  const selectedSessionId = useConversationStore((state) => state.selectedSessionId);
  const sessionsByProjectId = useConversationStore((state) => state.sessionsByProjectId);

  const isSettingsRoute = location.pathname === "/settings" || location.pathname.startsWith("/settings/");
  const selectedSession = useMemo(
    () => sessionsByProjectId[selectedProjectId ?? ""]?.find((item) => item.session_id === selectedSessionId),
    [selectedProjectId, selectedSessionId, sessionsByProjectId]
  );
  const title = isSettingsRoute ? t("nav.settings") : selectedSession?.title ?? t("conversation.newThread");

  const handleDragMouseDown = useCallback((event: MouseEvent<HTMLElement>) => {
    if (event.button !== 0) {
      return;
    }

    const target = event.target as HTMLElement;
    if (target.closest("[data-no-drag='true'],button,input,textarea,select,a,[role='button']")) {
      return;
    }

    void appWindow.startDragging();
  }, [appWindow]);

  return (
    <header
      className="flex h-11 shrink-0 items-center border-b border-border-subtle bg-background/95 select-none"
      data-tauri-drag-region
      onMouseDown={handleDragMouseDown}
    >
      <div
        className={cn(
          "flex h-full items-center gap-3 border-r border-border-subtle bg-muted/40 px-4",
          isSettingsRoute ? "w-sidebar" : sidebarCollapsed ? "w-sidebar-collapsed" : "w-sidebar"
        )}
      >
        <div
          className={cn("h-full shrink-0", isMac ? "w-14" : "w-2")}
          data-tauri-drag-region
          aria-hidden
        />
      </div>

      <div className="flex min-w-0 flex-1 items-center px-4" data-tauri-drag-region>
        <div className="min-w-0 flex-1">
          <p className="truncate text-small font-semibold text-foreground">{title}</p>
        </div>
      </div>
    </header>
  );
}
