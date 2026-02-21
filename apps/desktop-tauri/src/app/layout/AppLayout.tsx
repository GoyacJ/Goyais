import { useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { AppTopBar } from "@/app/layout/AppTopBar";
import { SettingsSidebar } from "@/app/layout/SettingsSidebar";
import { Sidebar } from "@/app/layout/Sidebar";
import { StatusBar } from "@/app/layout/StatusBar";
import { CommandPalette } from "@/components/domain/command/CommandPalette";
import { cn } from "@/lib/cn";
import { isEditableElement, isPaletteShortcut } from "@/lib/shortcuts";
import { useSettingsStore } from "@/stores/settingsStore";

export function AppLayout() {
  const location = useLocation();
  const theme = useSettingsStore((state) => state.theme);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const isSettingsRoute = location.pathname === "/settings" || location.pathname.startsWith("/settings/");

  useEffect(() => {
    const root = document.documentElement;
    if (theme === "light") {
      root.classList.add("light");
      root.classList.remove("dark");
    } else {
      root.classList.add("dark");
      root.classList.remove("light");
    }
  }, [theme]);

  useEffect(() => {
    const onKeydown = (event: KeyboardEvent) => {
      if (!isPaletteShortcut(event)) return;
      if (isEditableElement(event.target)) return;
      event.preventDefault();
      setPaletteOpen(true);
    };

    window.addEventListener("keydown", onKeydown);
    return () => window.removeEventListener("keydown", onKeydown);
  }, []);

  return (
    <div className="flex h-full flex-col bg-background text-foreground">
      <AppTopBar />

      <div className="flex min-h-0 flex-1">
        {isSettingsRoute ? <SettingsSidebar /> : <Sidebar />}
        <div className="flex min-w-0 flex-1 flex-col">
          <main
            className={cn(
              "min-h-0 flex-1 overflow-auto scrollbar-subtle",
              "p-0"
            )}
          >
            <Outlet />
          </main>
          <StatusBar />
        </div>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
