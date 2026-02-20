import { Code2, FolderKanban, History, Play, Settings2, SlidersHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import { useUiStore } from "@/stores/uiStore";

const navItems = [
  { to: "/", labelKey: "app.nav.run", icon: Play },
  { to: "/projects", labelKey: "app.nav.projects", icon: FolderKanban },
  { to: "/models", labelKey: "app.nav.models", icon: SlidersHorizontal },
  { to: "/replay", labelKey: "app.nav.replay", icon: History },
  { to: "/settings", labelKey: "app.nav.settings", icon: Settings2 }
];

export function Sidebar() {
  const { t } = useTranslation();
  const location = useLocation();
  const collapsed = useUiStore((state) => state.sidebarCollapsed);
  const toggleSidebar = useUiStore((state) => state.toggleSidebar);

  return (
    <aside
      className={cn(
        "border-r border-border-subtle bg-muted/40 p-3 transition-[width] duration-150",
        collapsed ? "w-sidebar-collapsed" : "w-sidebar"
      )}
    >
      <div className="mb-4 flex h-toolbar items-center justify-between gap-2">
        <div className="flex items-center gap-2 overflow-hidden">
          <Code2 className="h-4 w-4 text-accent" />
          {!collapsed ? <span className="truncate text-small font-semibold text-foreground">{t("app.brand")}</span> : null}
        </div>
        <Button
          size="sm"
          variant="ghost"
          onClick={toggleSidebar}
          aria-label={collapsed ? t("app.sidebar.expand") : t("app.sidebar.collapse")}
        >
          {collapsed ? ">" : "<"}
        </Button>
      </div>

      <nav className="space-y-1">
        {navItems.map((item) => {
          const label = t(item.labelKey);
          const active = location.pathname === item.to;
          const content = (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                "flex h-9 items-center gap-2 rounded-control px-2 text-small transition-colors",
                active ? "bg-accent/20 text-accent" : "text-muted-foreground hover:bg-muted hover:text-foreground",
                collapsed ? "justify-center px-0" : "justify-start"
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {!collapsed ? <span className="truncate">{label}</span> : null}
            </Link>
          );

          if (!collapsed) {
            return content;
          }

          return (
            <Tooltip key={item.to}>
              <TooltipTrigger asChild>{content}</TooltipTrigger>
              <TooltipContent side="right">{label}</TooltipContent>
            </Tooltip>
          );
        })}
      </nav>
    </aside>
  );
}
