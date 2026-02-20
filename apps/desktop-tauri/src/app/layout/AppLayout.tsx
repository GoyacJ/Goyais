import { useEffect, useState } from "react";
import { Outlet } from "react-router-dom";

import { Sidebar } from "@/app/layout/Sidebar";
import { Topbar } from "@/app/layout/Topbar";
import { CommandPalette } from "@/components/domain/command/CommandPalette";
import { isEditableElement, isPaletteShortcut } from "@/lib/shortcuts";
import { useUiStore } from "@/stores/uiStore";

export function AppLayout() {
  const theme = useUiStore((state) => state.theme);
  const [paletteOpen, setPaletteOpen] = useState(false);

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
    <div className="flex h-full bg-background text-foreground">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="min-h-0 flex-1 overflow-auto p-page scrollbar-subtle">
          <Outlet />
        </main>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
