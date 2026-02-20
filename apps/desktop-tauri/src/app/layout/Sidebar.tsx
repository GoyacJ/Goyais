import type { LucideIcon } from "lucide-react";
import {
  Circle,
  Code2,
  FolderKanban,
  History,
  Play,
  Settings2,
  SlidersHorizontal,
  TerminalSquare
} from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";

import { WorkspaceSwitcher } from "@/app/layout/WorkspaceSwitcher";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import { useUiStore } from "@/stores/uiStore";
import {
  type RemoteNavigationMenu,
  selectCurrentNavigation,
  selectCurrentWorkspaceKind,
  useWorkspaceStore} from "@/stores/workspaceStore";

interface SidebarNavItem {
  to: string;
  labelKey: string;
  icon: LucideIcon;
  depth: number;
}

const iconMap: Record<string, LucideIcon> = {
  folder: FolderKanban,
  terminal: TerminalSquare,
  clock: History,
  cpu: SlidersHorizontal,
  settings: Settings2
};

const localNavItems: SidebarNavItem[] = [
  { to: "/run", labelKey: "nav.run", icon: Play, depth: 0 },
  { to: "/projects", labelKey: "nav.projects", icon: FolderKanban, depth: 0 },
  { to: "/models", labelKey: "nav.models", icon: SlidersHorizontal, depth: 0 },
  { to: "/replay", labelKey: "nav.replay", icon: History, depth: 0 },
  { to: "/settings", labelKey: "nav.settings", icon: Settings2, depth: 0 }
];

function flattenRemoteMenus(menus: RemoteNavigationMenu[], depth = 0): SidebarNavItem[] {
  const items: SidebarNavItem[] = [];

  for (const menu of menus) {
    items.push({
      to: menu.route ?? "/settings",
      labelKey: menu.i18n_key,
      icon: menu.icon_key ? iconMap[menu.icon_key] ?? Circle : Circle,
      depth
    });

    if (menu.children.length > 0) {
      items.push(...flattenRemoteMenus(menu.children, depth + 1));
    }
  }

  return items;
}

export function Sidebar() {
  const { t } = useTranslation();
  const location = useLocation();
  const collapsed = useUiStore((state) => state.sidebarCollapsed);
  const toggleSidebar = useUiStore((state) => state.toggleSidebar);
  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);
  const currentNavigation = useWorkspaceStore(selectCurrentNavigation);

  const navItems = useMemo(() => {
    if (workspaceKind === "remote") {
      return flattenRemoteMenus(currentNavigation?.menus ?? []);
    }
    return localNavItems;
  }, [workspaceKind, currentNavigation]);

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
        <button
          type="button"
          className="inline-flex h-8 w-8 items-center justify-center rounded-control text-small text-muted-foreground hover:bg-muted"
          onClick={toggleSidebar}
          aria-label={collapsed ? t("app.sidebar.expand") : t("app.sidebar.collapse")}
        >
          {collapsed ? ">" : "<"}
        </button>
      </div>

      <div className="mb-3">
        <WorkspaceSwitcher collapsed={collapsed} />
      </div>

      {workspaceKind === "remote" && !currentNavigation ? (
        <div className="space-y-2 px-1 py-2">
          <div className="h-8 animate-pulse rounded-control bg-muted" />
          <div className="h-8 animate-pulse rounded-control bg-muted" />
          <div className="h-8 animate-pulse rounded-control bg-muted" />
        </div>
      ) : (
        <nav className="space-y-1">
          {navItems.map((item) => {
            const label = t(item.labelKey);
            const active = location.pathname === item.to || (item.to === "/run" && location.pathname === "/");
            const content = (
              <Link
                key={`${item.to}:${item.depth}`}
                to={item.to}
                className={cn(
                  "flex h-9 items-center gap-2 rounded-control px-2 text-small transition-colors",
                  active ? "bg-accent/20 text-accent" : "text-muted-foreground hover:bg-muted hover:text-foreground",
                  collapsed ? "justify-center px-0" : "justify-start"
                )}
                style={!collapsed && item.depth > 0 ? { paddingLeft: `${item.depth * 12 + 8}px` } : undefined}
              >
                <item.icon className="h-4 w-4 shrink-0" />
                {!collapsed ? <span className="truncate">{label}</span> : null}
              </Link>
            );

            if (!collapsed) {
              return content;
            }

            return (
              <Tooltip key={`${item.to}:${item.depth}`}>
                <TooltipTrigger asChild>{content}</TooltipTrigger>
                <TooltipContent side="right">{label}</TooltipContent>
              </Tooltip>
            );
          })}
        </nav>
      )}
    </aside>
  );
}
