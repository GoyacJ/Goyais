import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { EventType } from "@/types/generated";

const eventTypeOrder: EventType[] = ["plan", "tool_call", "tool_result", "patch", "error", "done"];

interface TimelineFiltersProps {
  filters: Record<EventType, boolean>;
  autoFollow: boolean;
  onToggleFilter: (eventType: EventType) => void;
  onToggleAutoFollow: () => void;
}

export function TimelineFilters({ filters, autoFollow, onToggleFilter, onToggleAutoFollow }: TimelineFiltersProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-border-subtle pb-2">
      {eventTypeOrder.map((type) => (
        <button key={type} type="button" onClick={() => onToggleFilter(type)}>
          <Badge variant={filters[type] ? "default" : "secondary"}>
            {t(`timeline.types.${type}`, { defaultValue: type })}
          </Badge>
        </button>
      ))}
      <div className="ml-auto">
        <Button size="sm" variant={autoFollow ? "default" : "outline"} onClick={onToggleAutoFollow}>
          {t("timeline.autoFollow", {
            state: autoFollow ? t("timeline.state.on") : t("timeline.state.off")
          })}
        </Button>
      </div>
    </div>
  );
}
