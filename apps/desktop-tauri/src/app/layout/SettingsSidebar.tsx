import { ArrowLeft, Bot, Gauge, Plug, Sparkles, SlidersHorizontal } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";

import { cn } from "@/lib/cn";

const SETTINGS_SECTIONS = [
  { id: "general", icon: SlidersHorizontal },
  { id: "runtime", icon: Gauge },
  { id: "models", icon: Bot },
  { id: "skills", icon: Sparkles },
  { id: "mcp", icon: Plug }
] as const;

type SettingsSectionKey = (typeof SETTINGS_SECTIONS)[number]["id"];

function asSection(pathname: string): SettingsSectionKey {
  const section = pathname.split("/")[2] ?? "general";
  if (section === "runtime" || section === "models" || section === "skills" || section === "mcp") {
    return section;
  }
  return "general";
}

export function SettingsSidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const activeSection = useMemo(() => asSection(location.pathname), [location.pathname]);

  return (
    <aside className="flex h-full w-sidebar flex-col border-r border-border-subtle bg-muted/40 p-3">
      <button
        type="button"
        className="mb-4 flex items-center gap-2 rounded-control px-2 py-2 text-left text-small text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        onClick={() => navigate("/")}
      >
        <ArrowLeft className="h-4 w-4" />
        <span>{t("settings.backToApp")}</span>
      </button>

      <nav className="space-y-1">
        {SETTINGS_SECTIONS.map((section) => (
          <button
            key={section.id}
            type="button"
            className={cn(
              "flex w-full items-center gap-2 rounded-control px-2 py-2 text-left text-small transition-colors",
              activeSection === section.id
                ? "bg-muted text-foreground"
                : "text-muted-foreground hover:bg-muted/70 hover:text-foreground"
            )}
            onClick={() => navigate(`/settings/${section.id}`)}
          >
            <section.icon className="h-4 w-4" />
            <div className="min-w-0">
              <p className="truncate">{t(`settings.sections.${section.id}`)}</p>
              <p className="truncate text-xs text-muted-foreground">{t(`settings.sections.${section.id}Description`)}</p>
            </div>
          </button>
        ))}
      </nav>
    </aside>
  );
}
